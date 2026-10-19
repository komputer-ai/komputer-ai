package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// The CRD itself refuses lifecycle: Archive on a schedule's agent template, so a
// schedule written with kubectl can't get there either (the API checks too).
var _ = Describe("KomputerSchedule CRD validation", func() {
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

	It("rejects an Archive agent template", func() {
		err := k8sClient.Create(context.Background(), schedule("archive-sched", komputerv1alpha1.AgentLifecycleArchive))
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("Archive isn't supported for schedules"))
	})

	It("accepts an AutoDelete agent template", func() {
		s := schedule("autodelete-sched", komputerv1alpha1.AgentLifecycleAutoDelete)
		Expect(k8sClient.Create(context.Background(), s)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), s)).To(Succeed())
	})
})
