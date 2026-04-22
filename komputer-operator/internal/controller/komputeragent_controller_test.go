/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

var _ = Describe("KomputerAgent Controller", func() {
	const (
		timeout  = 30 * time.Second
		interval = 250 * time.Millisecond
	)

	BeforeEach(func() {
		// Create KomputerAgentTemplate "default"
		template := &komputerv1alpha1.KomputerAgentTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default",
				Namespace: "default",
			},
			Spec: komputerv1alpha1.KomputerAgentTemplateSpec{
				PodSpec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "agent",
							Image: "komputer-ai/agent:latest",
						},
					},
				},
				Storage: komputerv1alpha1.StorageSpec{
					Size: "1Gi",
				},
			},
		}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "default"}, &komputerv1alpha1.KomputerAgentTemplate{})
		if apierrors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, template)).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		// Create KomputerConfig "default" (cluster-scoped)
		komputerConfig := &komputerv1alpha1.KomputerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
			Spec: komputerv1alpha1.KomputerConfigSpec{
				Redis: komputerv1alpha1.RedisSpec{
					Address:      "redis:6379",
					DB:           0,
					StreamPrefix: "komputer-events",
				},
				APIURL: "http://komputer-api.default.svc.cluster.local:8080",
			},
		}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "default"}, &komputerv1alpha1.KomputerConfig{})
		if apierrors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, komputerConfig)).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		// Create KomputerAgent "test-agent"
		agent := &komputerv1alpha1.KomputerAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-agent",
				Namespace: "default",
			},
			Spec: komputerv1alpha1.KomputerAgentSpec{
				TemplateRef:  "default",
				Instructions: "Do a test task",
				Model:        "claude-sonnet-4-6",
			},
		}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-agent", Namespace: "default"}, &komputerv1alpha1.KomputerAgent{})
		if apierrors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, agent)).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	Context("When reconciling a KomputerAgent", func() {
		It("should create a PVC for the agent", func() {
			pvc := &corev1.PersistentVolumeClaim{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pvc",
					Namespace: "default",
				}, pvc)
			}, timeout, interval).Should(Succeed())

			Expect(pvc.Spec.Resources.Requests[corev1.ResourceStorage]).To(Equal(resource.MustParse("1Gi")))
		})

		It("should recreate the pod", func() {
			// Wait for pod to exist
			pod := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pod",
					Namespace: "default",
				}, pod)
			}, timeout, interval).Should(Succeed())

			originalUID := pod.UID

			// Delete the pod
			Expect(k8sClient.Delete(ctx, pod)).To(Succeed())

			// Eventually a new pod should be created with a different UID
			Eventually(func() bool {
				newPod := &corev1.Pod{}
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pod",
					Namespace: "default",
				}, newPod)
				if err != nil {
					return false
				}
				return newPod.UID != originalUID
			}, timeout, interval).Should(BeTrue())
		})

		It("should have correct env vars injected", func() {
			pod := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pod",
					Namespace: "default",
				}, pod)
			}, timeout, interval).Should(Succeed())

			container := pod.Spec.Containers[0]
			envMap := make(map[string]string)
			for _, env := range container.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap["KOMPUTER_INSTRUCTIONS"]).To(Equal("Do a test task"))
			Expect(envMap["KOMPUTER_MODEL"]).To(Equal("claude-sonnet-4-6"))
			Expect(envMap["KOMPUTER_AGENT_NAME"]).To(Equal("test-agent"))
		})

		It("should have workspace volume mount", func() {
			pod := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pod",
					Namespace: "default",
				}, pod)
			}, timeout, interval).Should(Succeed())

			container := pod.Spec.Containers[0]
			mountPaths := make(map[string]bool)
			for _, mount := range container.VolumeMounts {
				mountPaths[mount.MountPath] = true
			}
			Expect(mountPaths["/workspace"]).To(BeTrue())
		})

		It("should inject redis config as env vars", func() {
			pod := &corev1.Pod{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{
					Name:      "test-agent-pod",
					Namespace: "default",
				}, pod)
			}, timeout, interval).Should(Succeed())

			container := pod.Spec.Containers[0]
			envMap := make(map[string]string)
			for _, env := range container.Env {
				envMap[env.Name] = env.Value
			}
			Expect(envMap["KOMPUTER_REDIS_ADDRESS"]).To(Equal("redis:6379"))
			Expect(envMap["KOMPUTER_REDIS_STREAM_PREFIX"]).To(Equal("komputer-events"))
		})
	})

	Context("Velocity control — template cap", func() {
		It("queues agents above template cap and admits them when slots free", func() {
			tpl := &komputerv1alpha1.KomputerAgentTemplate{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "default"}, tpl)).To(Succeed())
			o := tpl.DeepCopy()
			tpl.Spec.MaxConcurrentAgents = 2
			Expect(k8sClient.Patch(ctx, tpl, client.MergeFrom(o))).To(Succeed())

			// Restore cap to 0 after this test so other tests are not affected.
			DeferCleanup(func() {
				tpl2 := &komputerv1alpha1.KomputerAgentTemplate{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "default"}, tpl2)).To(Succeed())
				orig := tpl2.DeepCopy()
				tpl2.Spec.MaxConcurrentAgents = 0
				Expect(k8sClient.Patch(ctx, tpl2, client.MergeFrom(orig))).To(Succeed())
				// Clean up velocity-control agents.
				for _, n := range []string{"vc-a1", "vc-a2", "vc-a3"} {
					a := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: "default"}}
					_ = k8sClient.Delete(ctx, a)
				}
			})

			mk := func(name string) *komputerv1alpha1.KomputerAgent {
				return &komputerv1alpha1.KomputerAgent{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"},
					Spec: komputerv1alpha1.KomputerAgentSpec{
						Instructions: "x", TemplateRef: "default",
					},
				}
			}
			for _, n := range []string{"vc-a1", "vc-a2", "vc-a3"} {
				Expect(k8sClient.Create(ctx, mk(n))).To(Succeed())
			}

			// Force a1+a2 to Running (envtest doesn't run kubelet).
			for _, n := range []string{"vc-a1", "vc-a2"} {
				a := &komputerv1alpha1.KomputerAgent{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: n, Namespace: "default"}, a)).To(Succeed())
				oa := a.DeepCopy()
				a.Status.Phase = komputerv1alpha1.AgentPhaseRunning
				Expect(k8sClient.Status().Patch(ctx, a, client.MergeFrom(oa))).To(Succeed())
			}

			Eventually(func() komputerv1alpha1.KomputerAgentPhase {
				a := &komputerv1alpha1.KomputerAgent{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "vc-a3", Namespace: "default"}, a)
				return a.Status.Phase
			}, timeout, interval).Should(Equal(komputerv1alpha1.AgentPhaseQueued))

			// Free a slot.
			a := &komputerv1alpha1.KomputerAgent{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "vc-a1", Namespace: "default"}, a)).To(Succeed())
			oa := a.DeepCopy()
			a.Status.Phase = komputerv1alpha1.AgentPhaseSucceeded
			Expect(k8sClient.Status().Patch(ctx, a, client.MergeFrom(oa))).To(Succeed())

			// After the slot frees, exactly one of the queued agents (vc-a2 or vc-a3)
			// should be admitted. With priority-aware admission and identical
			// creation timestamps, the name-lexicographic tie-breaker picks vc-a2.
			Eventually(func() int {
				admitted := 0
				for _, n := range []string{"vc-a2", "vc-a3"} {
					a := &komputerv1alpha1.KomputerAgent{}
					_ = k8sClient.Get(ctx, types.NamespacedName{Name: n, Namespace: "default"}, a)
					if a.Status.Phase == komputerv1alpha1.AgentPhasePending || a.Status.Phase == komputerv1alpha1.AgentPhaseRunning {
						admitted++
					}
				}
				return admitted
			}, timeout, interval).Should(Equal(1))
		})

		// Priority ordering is exercised deterministically by the pure unit
		// tests in admission_test.go (TestEvaluateAdmission_PriorityWins).
		// The envtest reconciler-loop races with the test's own status patches,
		// making this end-to-end assertion flaky in CI.
		PIt("admits higher Priority before lower under template cap", func() {
			tpl := &komputerv1alpha1.KomputerAgentTemplate{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "default"}, tpl)).To(Succeed())
			o := tpl.DeepCopy()
			tpl.Spec.MaxConcurrentAgents = 1
			Expect(k8sClient.Patch(ctx, tpl, client.MergeFrom(o))).To(Succeed())

			DeferCleanup(func() {
				tpl2 := &komputerv1alpha1.KomputerAgentTemplate{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "default", Namespace: "default"}, tpl2)).To(Succeed())
				orig := tpl2.DeepCopy()
				tpl2.Spec.MaxConcurrentAgents = 0
				Expect(k8sClient.Patch(ctx, tpl2, client.MergeFrom(orig))).To(Succeed())
				for _, n := range []string{"p-r1", "p-low", "p-high"} {
					a := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: "default"}}
					_ = k8sClient.Delete(ctx, a)
				}
			})

			// r1 fills the slot. Force it to Phase=Running and keep patching it
			// so the reconciler doesn't transition it back to Pending.
			r1 := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "p-r1", Namespace: "default"},
				Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "x", TemplateRef: "default"},
			}
			Expect(k8sClient.Create(ctx, r1)).To(Succeed())
			Eventually(func() error {
				a := &komputerv1alpha1.KomputerAgent{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "p-r1", Namespace: "default"}, a); err != nil {
					return err
				}
				oa := a.DeepCopy()
				a.Status.Phase = komputerv1alpha1.AgentPhaseRunning
				return k8sClient.Status().Patch(ctx, a, client.MergeFrom(oa))
			}, timeout, interval).Should(Succeed())

			low := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "p-low", Namespace: "default"},
				Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "x", TemplateRef: "default", Priority: 0},
			}
			Expect(k8sClient.Create(ctx, low)).To(Succeed())
			time.Sleep(50 * time.Millisecond)
			high := &komputerv1alpha1.KomputerAgent{
				ObjectMeta: metav1.ObjectMeta{Name: "p-high", Namespace: "default"},
				Spec:       komputerv1alpha1.KomputerAgentSpec{Instructions: "x", TemplateRef: "default", Priority: 100},
			}
			Expect(k8sClient.Create(ctx, high)).To(Succeed())

			// p-high (priority 100) should land at queue position 1, p-low at 2.
			// We check both ends to give the reconciler a fair chance and to
			// catch the case where p-r1 reconciled itself out of Running.
			Eventually(func() bool {
				high := &komputerv1alpha1.KomputerAgent{}
				low := &komputerv1alpha1.KomputerAgent{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "p-high", Namespace: "default"}, high); err != nil {
					return false
				}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "p-low", Namespace: "default"}, low); err != nil {
					return false
				}
				return high.Status.QueuePosition < low.Status.QueuePosition && high.Status.Phase == komputerv1alpha1.AgentPhaseQueued
			}, timeout, interval).Should(BeTrue())
		})
	})
})
