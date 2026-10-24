package controller

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func saverTestObjects() (*komputerv1alpha1.KomputerAgentTemplate, *komputerv1alpha1.KomputerAgent) {
	nonRoot, noEscalation := true, false
	uid := int64(1000)
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			ServiceAccountName: "agents",
			SecurityContext:    &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot, RunAsUser: &uid, FSGroup: &uid},
			ImagePullSecrets:   []corev1.LocalObjectReference{{Name: "registry"}},
			NodeSelector:       map[string]string{"pool": "agents"},
			Tolerations:        []corev1.Toleration{{Key: "agents", Operator: corev1.TolerationOpExists}},
			Containers: []corev1.Container{{
				Name: "agent", Image: "agent-img:1",
				SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &noEscalation,
					Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
					Limits:   corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")},
				},
				Env: []corev1.EnvVar{{Name: "CLAUDE_CODE_USE_BEDROCK", Value: "1"}},
			}},
		}},
	}
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", UID: types.UID("u-1")},
		Spec: komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{
			// The agent's own overrides must reach the saver pod too.
			PodSpec: &corev1.PodSpec{
				NodeSelector: map[string]string{"pool": "big"},
				Containers: []corev1.Container{{Name: "agent", Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("4Gi")},
				}}},
			},
		}},
		Status: komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseSleeping, TaskStatus: komputerv1alpha1.AgentTaskComplete},
	}
	return tpl, agent
}

// normalize clears exactly what a saver pod may change on purpose.
func normalize(spec corev1.PodSpec) corev1.PodSpec {
	s := *spec.DeepCopy()
	c := &s.Containers[0]
	c.Command, c.Args, c.LivenessProbe, c.ReadinessProbe, c.Lifecycle = nil, nil, nil, nil, nil
	for i := range c.VolumeMounts {
		c.VolumeMounts[i].ReadOnly = false
	}
	for i := range s.Volumes {
		if s.Volumes[i].PersistentVolumeClaim != nil {
			s.Volumes[i].PersistentVolumeClaim.ReadOnly = false
		}
	}
	s.ActiveDeadlineSeconds = nil
	return s
}

// The saver pod copies the agent's (long) grace period, and its sleep ignores
// SIGTERM: once the copy-out is done it's deleted without waiting that out, so
// it doesn't hold a saver slot for a minute.
func TestSessionFinalizer_DeletesTheSaverPodWithoutWaitingForGrace(t *testing.T) {
	agent := deletingAgent(func(a *komputerv1alpha1.KomputerAgent) {
		a.Status.Phase = komputerv1alpha1.AgentPhaseSleeping
		a.Status.PodName = ""
	})
	saver := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "a1-session-saver", Namespace: "default", Labels: map[string]string{komputerv1alpha1.LabelSessionSaver: "true"}},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "agent"}}},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	var grace *int64
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(agent, agentPVC(false), saver, agentTemplate(), s3KomputerConfig()).
		WithStatusSubresource(&komputerv1alpha1.KomputerAgent{}).
		WithInterceptorFuncs(interceptor.Funcs{Delete: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.DeleteOption) error {
			if obj.GetName() == "a1-session-saver" {
				do := &client.DeleteOptions{}
				do.ApplyOptions(opts)
				grace = do.GracePeriodSeconds
			}
			return cl.Delete(ctx, obj, opts...)
		}}).Build()
	r := &KomputerAgentReconciler{Client: c, Scheme: scheme, Exec: &fakeExecer{out: []byte("tar")}, MaxConcurrentSavers: 4,
		NewUploader: func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (SessionUploader, error) {
			return &fakeUploader{}, nil
		}}

	if _, err := r.handleSessionFinalizer(context.Background(), agent); err != nil {
		t.Fatal(err)
	}
	if grace == nil || *grace > 1 {
		t.Errorf("saver pod deleted with grace %v, want at most 1s", grace)
	}
}

func TestBuildSaverPod_MatchesTheAgentPod(t *testing.T) {
	tpl, agent := saverTestObjects()
	cfg := s3KomputerConfig()
	r, _ := saveFirstReconciler(t, &fakeExecer{}, &fakeUploader{}, tpl, agent, cfg)

	saver, err := r.buildSaverPod(context.Background(), agent, "a1-pvc")
	if err != nil {
		t.Fatal(err)
	}
	agentPod, err := r.buildPod(context.Background(), agent, applyAgentOverrides(tpl, agent), "a1-pvc", "a1-pod", cfg, false)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(normalize(saver.Spec), normalize(agentPod.Spec)) {
		t.Errorf("saver pod spec differs from the agent's\n saver: %+v\n agent: %+v", normalize(saver.Spec), normalize(agentPod.Spec))
	}

	// The saver-only differences: it doesn't run the agent, mounts read-only, has a deadline,
	// and isn't selected by the agent's Service.
	c := saver.Spec.Containers[0]
	if len(c.Command) == 0 || c.Command[0] != "sleep" || c.LivenessProbe != nil || c.ReadinessProbe != nil || c.Lifecycle != nil {
		t.Errorf("saver container must only sleep: command=%v probes=%v/%v lifecycle=%v", c.Command, c.LivenessProbe, c.ReadinessProbe, c.Lifecycle)
	}
	for _, v := range saver.Spec.Volumes {
		if v.PersistentVolumeClaim != nil && !v.PersistentVolumeClaim.ReadOnly {
			t.Errorf("workspace volume %s must be read-only", v.Name)
		}
	}
	if saver.Spec.ActiveDeadlineSeconds == nil || *saver.Spec.ActiveDeadlineSeconds != sessionSaverDeadline {
		t.Errorf("deadline = %v", saver.Spec.ActiveDeadlineSeconds)
	}
	if saver.Labels[komputerv1alpha1.LabelSessionSaver] != "true" || saver.Labels["komputer.ai/member-a1"] != "" {
		t.Errorf("labels = %v: want the saver label and not the Service's member label", saver.Labels)
	}
}
