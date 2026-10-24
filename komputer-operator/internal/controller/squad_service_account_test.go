package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func saTemplate(name, sa string) *komputerv1alpha1.KomputerAgentTemplate {
	return &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			ServiceAccountName: sa,
			Containers:         []corev1.Container{{Name: "agent", Image: "img"}},
		}},
	}
}

func saMember(name, templateRef string, port int32) *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{TemplateRef: templateRef}},
		Status:     komputerv1alpha1.KomputerAgentStatus{Port: port, PvcName: name + "-pvc"},
	}
}

func TestBuildSquadPodSpec_ServiceAccount(t *testing.T) {
	scheme := newTestScheme(t)
	for _, tc := range []struct {
		name        string
		templates   []*komputerv1alpha1.KomputerAgentTemplate
		refs        [2]string
		wantSA      string
		wantWarning bool
	}{
		{"members agree", []*komputerv1alpha1.KomputerAgentTemplate{saTemplate("t1", "x")}, [2]string{"t1", "t1"}, "x", false},
		{"members differ", []*komputerv1alpha1.KomputerAgentTemplate{saTemplate("t1", "x"), saTemplate("t2", "y")}, [2]string{"t1", "t2"}, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme)
			for _, tpl := range tc.templates {
				builder = builder.WithObjects(tpl)
			}
			r := &KomputerSquadReconciler{Client: builder.Build(), Scheme: scheme}
			agents := []*komputerv1alpha1.KomputerAgent{saMember("m1", tc.refs[0], 8000), saMember("m2", tc.refs[1], 8001)}

			pod, err := r.buildSquadPodSpec(context.Background(), testSquad("m1", "m2"), agents, &komputerv1alpha1.KomputerConfig{})
			if err != nil {
				t.Fatalf("buildSquadPodSpec: %v", err)
			}
			if pod.Spec.ServiceAccountName != tc.wantSA {
				t.Errorf("ServiceAccountName = %q, want %q", pod.Spec.ServiceAccountName, tc.wantSA)
			}
			sa, mismatched := r.squadServiceAccount(context.Background(), agents)
			if sa != tc.wantSA || mismatched != tc.wantWarning {
				t.Errorf("squadServiceAccount = (%q, %v), want (%q, %v)", sa, mismatched, tc.wantSA, tc.wantWarning)
			}
		})
	}
}

func TestBuildSquadPodSpec_S3SetsGracePeriod(t *testing.T) {
	scheme := newTestScheme(t)
	r := &KomputerSquadReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(saTemplate("default", "")).Build(), Scheme: scheme}
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}}}
	pod, err := r.buildSquadPodSpec(context.Background(), testSquad("m1"), []*komputerv1alpha1.KomputerAgent{saMember("m1", "", 8000)}, config)
	if err != nil {
		t.Fatal(err)
	}
	if got := pod.Spec.TerminationGracePeriodSeconds; got == nil || *got != *ptr.To(int64(60)) {
		t.Errorf("terminationGracePeriodSeconds = %v, want 60", got)
	}
	if !podHasS3Env(pod.Spec.Containers[0]) {
		t.Error("squad member containers must carry the S3 env")
	}
}
