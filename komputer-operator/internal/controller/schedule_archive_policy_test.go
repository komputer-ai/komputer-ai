package controller

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// loadSchedulePolicyTemplate reads the Helm template for the schedule admission
// policy and drops the template directives, so the test exercises the exact CEL
// the chart ships rather than a copy of it.
func loadSchedulePolicyTemplate() []client.Object {
	path := filepath.Join("..", "..", "..", "helm", "komputer-ai", "templates", "komputer-operator", "schedule-archive-policy.yaml")
	raw, err := os.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())

	var kept []string
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.Contains(line, "{{") {
			kept = append(kept, line)
		}
	}
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	var objs []client.Object
	for _, doc := range strings.Split(strings.Join(kept, "\n"), "\n---\n") {
		obj, _, err := decoder.Decode([]byte(doc), nil, nil)
		Expect(err).NotTo(HaveOccurred())
		objs = append(objs, obj.(client.Object))
	}
	return objs
}

var _ = Describe("Schedule archive admission policy", func() {
	schedule := func(name string, lifecycle komputerv1alpha1.AgentLifecycle) *komputerv1alpha1.KomputerSchedule {
		return &komputerv1alpha1.KomputerSchedule{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
			Spec: komputerv1alpha1.KomputerScheduleSpec{
				Schedule:     "0 0 * * *",
				Instructions: "go",
				Agent:        &komputerv1alpha1.ScheduleAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: lifecycle}},
			},
		}
	}

	It("rejects an Archive schedule without the S3 session store and allows it with one", func() {
		ctx := context.Background()

		for _, obj := range loadSchedulePolicyTemplate() {
			Expect(k8sClient.Create(ctx, obj)).To(Succeed())
			DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), obj) })
		}

		cfg := &komputerv1alpha1.KomputerConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "policy-test"},
			Spec:       komputerv1alpha1.KomputerConfigSpec{Redis: komputerv1alpha1.RedisSpec{Address: "localhost:6379"}},
		}
		Expect(k8sClient.Create(ctx, cfg)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), cfg) })

		By("denying Archive while no KomputerConfig has S3")
		Eventually(func() string {
			err := k8sClient.Create(ctx, schedule("archive-no-s3", komputerv1alpha1.AgentLifecycleArchive))
			if err == nil {
				_ = k8sClient.Delete(ctx, schedule("archive-no-s3", ""))
				return ""
			}
			return err.Error()
		}, 20*time.Second, 500*time.Millisecond).Should(ContainSubstring("requires the S3 session store"))

		By("still allowing other lifecycles")
		s := schedule("sleep-no-s3", komputerv1alpha1.AgentLifecycleSleep)
		Expect(k8sClient.Create(ctx, s)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), s) })

		By("allowing Archive once every KomputerConfig has S3")
		var cfgs komputerv1alpha1.KomputerConfigList
		Expect(k8sClient.List(ctx, &cfgs)).To(Succeed())
		for i := range cfgs.Items {
			c := cfgs.Items[i]
			original := c.Spec.S3
			c.Spec.S3 = &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}
			Expect(k8sClient.Update(ctx, &c)).To(Succeed())
			DeferCleanup(func() {
				var cur komputerv1alpha1.KomputerConfig
				if k8sClient.Get(context.Background(), client.ObjectKeyFromObject(&c), &cur) == nil {
					cur.Spec.S3 = original
					_ = k8sClient.Update(context.Background(), &cur)
				}
			})
		}
		archived := schedule("archive-with-s3", komputerv1alpha1.AgentLifecycleArchive)
		Eventually(func() error { return k8sClient.Create(ctx, archived) }, 20*time.Second, 500*time.Millisecond).Should(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(context.Background(), archived) })
	})
})
