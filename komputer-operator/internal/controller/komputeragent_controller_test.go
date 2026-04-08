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
})
