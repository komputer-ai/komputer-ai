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
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// hasExistingWorkspacePVC is the data-safety backstop that stops an ephemeral
// agent from getting an empty emptyDir mounted over a real workspace. These pin
// each of its signals independently: status.pvcName alone is not enough, since
// status can be lost or rewritten while the PVC object still exists.

func TestHasExistingWorkspacePVC_PVCObjectWithoutStatus(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"},
		// Status.PvcName deliberately empty.
	}
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: metav1.ObjectMeta{Name: "a-pvc", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).WithObjects(pvc).Build()

	got, err := hasExistingWorkspacePVC(context.Background(), c, agent, "a-pvc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("a PVC object that exists must be kept even when status.pvcName is empty")
	}
}

func TestHasExistingWorkspacePVC_StatusWithoutPVCObject(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"},
		Status:     komputerv1alpha1.KomputerAgentStatus{PvcName: "a-pvc"},
	}
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	got, err := hasExistingWorkspacePVC(context.Background(), c, agent, "a-pvc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got {
		t.Fatal("status.pvcName must count as an existing workspace")
	}
}

func TestHasExistingWorkspacePVC_Neither(t *testing.T) {
	agent := &komputerv1alpha1.KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build()

	got, err := hasExistingWorkspacePVC(context.Background(), c, agent, "a-pvc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got {
		t.Fatal("a fresh agent with no PVC and no status must be free to use an emptyDir")
	}
}
