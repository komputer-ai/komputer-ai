package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func s3TestAgent() *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "team-a", UID: types.UID("u-1")},
		Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "go"},
	}
}

func envByName(env []corev1.EnvVar) map[string]corev1.EnvVar {
	m := map[string]corev1.EnvVar{}
	for _, e := range env {
		m[e.Name] = e
	}
	return m
}

func TestBuildS3EnvVars_Disabled_ReturnsNil(t *testing.T) {
	if got := buildS3EnvVars(nil, s3TestAgent()); got != nil {
		t.Errorf("nil spec: got %v, want nil", got)
	}
	if got := buildS3EnvVars(&komputerv1alpha1.S3Spec{Enabled: false, Bucket: "b"}, s3TestAgent()); got != nil {
		t.Errorf("disabled spec: got %v, want nil", got)
	}
}

func TestBuildS3EnvVars_StaticKeys(t *testing.T) {
	s3 := &komputerv1alpha1.S3Spec{
		Enabled: true, Bucket: "sessions", Region: "eu-west-1", Endpoint: "http://minio:9000",
		ForcePathStyle: true, Prefix: "komputer",
		CredentialsSecret: &komputerv1alpha1.S3CredentialsSecretRef{Name: "s3-creds", Namespace: "komputer-system"},
	}
	env := envByName(buildS3EnvVars(s3, s3TestAgent()))
	for name, want := range map[string]string{
		"KOMPUTER_S3_BUCKET":           "sessions",
		"KOMPUTER_S3_REGION":           "eu-west-1",
		"AWS_REGION":                   "eu-west-1",
		"KOMPUTER_S3_ENDPOINT":         "http://minio:9000",
		"KOMPUTER_S3_FORCE_PATH_STYLE": "true",
		"KOMPUTER_S3_KEY":              "komputer/team-a/a1/u-1/session.tar.gz",
	} {
		if got := env[name].Value; got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	for name, key := range map[string]string{
		"AWS_ACCESS_KEY_ID":     komputerv1alpha1.S3CredentialsAccessKeyKey,
		"AWS_SECRET_ACCESS_KEY": komputerv1alpha1.S3CredentialsSecretKeyKey,
	} {
		ref := env[name].ValueFrom
		if ref == nil || ref.SecretKeyRef == nil {
			t.Fatalf("%s: want a secretKeyRef, got %+v", name, env[name])
		}
		if ref.SecretKeyRef.Name != "s3-creds" || ref.SecretKeyRef.Key != key {
			t.Errorf("%s: secretKeyRef = %s/%s, want s3-creds/%s", name, ref.SecretKeyRef.Name, ref.SecretKeyRef.Key, key)
		}
	}
}

// A missing or broken credentials Secret must never stop agent pods from starting.
func TestBuildS3EnvVars_CredentialRefsAreOptional(t *testing.T) {
	s3 := &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b", CredentialsSecret: &komputerv1alpha1.S3CredentialsSecretRef{Name: "s3-creds"}}
	env := envByName(buildS3EnvVars(s3, s3TestAgent()))
	for _, name := range []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		ref := env[name].ValueFrom.SecretKeyRef
		if ref.Optional == nil || !*ref.Optional {
			t.Errorf("%s: secretKeyRef.optional must be true", name)
		}
	}
}

func TestBuildS3EnvVars_IAMRole_NoCredentialVars(t *testing.T) {
	env := envByName(buildS3EnvVars(&komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}, s3TestAgent()))
	if _, ok := env["AWS_ACCESS_KEY_ID"]; ok {
		t.Error("no credentials Secret configured: AWS_ACCESS_KEY_ID must not be set")
	}
	if env["KOMPUTER_S3_FORCE_PATH_STYLE"].Value != "false" {
		t.Errorf("KOMPUTER_S3_FORCE_PATH_STYLE = %q, want false", env["KOMPUTER_S3_FORCE_PATH_STYLE"].Value)
	}
}

func TestBuildAgentEnvVars_IncludesS3WhenConfigured(t *testing.T) {
	scheme := newTestScheme(t)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}}}
	env, err := buildAgentEnvVars(context.Background(), c, s3TestAgent(), config)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := envByName(env)["KOMPUTER_S3_KEY"]; !ok {
		t.Error("buildAgentEnvVars must include the S3 env when S3 is configured")
	}
}

func TestPodHasS3Env(t *testing.T) {
	if podHasS3Env(corev1.Container{}) {
		t.Error("container without KOMPUTER_S3_KEY")
	}
	if !podHasS3Env(corev1.Container{Env: []corev1.EnvVar{{Name: "KOMPUTER_S3_KEY", Value: "k"}}}) {
		t.Error("container with KOMPUTER_S3_KEY")
	}
}

func TestReconcileAgentSecrets_MirrorsS3CredentialsSecret(t *testing.T) {
	scheme := newTestScheme(t)
	src := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "s3-creds", Namespace: "komputer-system"},
		Data:       map[string][]byte{"AWS_ACCESS_KEY_ID": []byte("ak"), "AWS_SECRET_ACCESS_KEY": []byte("sk")},
	}
	tpl := &komputerv1alpha1.KomputerAgentTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "team-a"},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(src, tpl).Build()
	config := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: &komputerv1alpha1.S3Spec{
		Enabled: true, Bucket: "b", CredentialsSecret: &komputerv1alpha1.S3CredentialsSecretRef{Name: "s3-creds"},
	}}}
	if _, err := reconcileAgentSecrets(context.Background(), c, s3TestAgent(), config, "komputer-system"); err != nil {
		t.Fatalf("reconcileAgentSecrets: %v", err)
	}
	var mirrored corev1.Secret
	if err := c.Get(context.Background(), types.NamespacedName{Name: "s3-creds", Namespace: "team-a"}, &mirrored); err != nil {
		t.Fatalf("S3 credentials Secret should be mirrored into the agent namespace: %v", err)
	}
	if string(mirrored.Data["AWS_ACCESS_KEY_ID"]) != "ak" {
		t.Errorf("mirrored data = %v", mirrored.Data)
	}
}

func TestBuildPod_S3SetsGracePeriodUnlessTemplateSets(t *testing.T) {
	scheme := newTestScheme(t)
	r := &KomputerAgentReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).Build(), Scheme: scheme}
	on := &komputerv1alpha1.KomputerConfig{Spec: komputerv1alpha1.KomputerConfigSpec{S3: &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}}}
	off := &komputerv1alpha1.KomputerConfig{}
	template := func(grace *int64) *komputerv1alpha1.KomputerAgentTemplate {
		return &komputerv1alpha1.KomputerAgentTemplate{Spec: komputerv1alpha1.KomputerAgentTemplateSpec{PodSpec: corev1.PodSpec{
			TerminationGracePeriodSeconds: grace,
			Containers:                    []corev1.Container{{Name: "agent", Image: "img"}},
		}}}
	}
	for _, tc := range []struct {
		name   string
		config *komputerv1alpha1.KomputerConfig
		tpl    *int64
		want   *int64
	}{
		{"S3 on, template unset", on, nil, ptr.To(int64(60))},
		{"S3 on, template sets 15", on, ptr.To(int64(15)), ptr.To(int64(15))},
		{"S3 off", off, nil, nil},
	} {
		pod, err := r.buildPod(context.Background(), s3TestAgent(), template(tc.tpl), "a1-pvc", "a1-pod", tc.config, false)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		got := pod.Spec.TerminationGracePeriodSeconds
		if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
			t.Errorf("%s: terminationGracePeriodSeconds = %v, want %v", tc.name, got, tc.want)
		}
	}
}
