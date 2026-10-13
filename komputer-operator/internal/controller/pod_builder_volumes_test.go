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
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestBuildAgentVolumes_PVC(t *testing.T) {
	vols := buildAgentVolumes("workspace", "my-agent-pvc", false, nil)
	if len(vols) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(vols))
	}
	v := vols[0]
	if v.Name != "workspace" {
		t.Errorf("name = %q, want workspace", v.Name)
	}
	if v.PersistentVolumeClaim == nil {
		t.Fatal("expected a PersistentVolumeClaim volume source")
	}
	if v.PersistentVolumeClaim.ClaimName != "my-agent-pvc" {
		t.Errorf("claimName = %q, want my-agent-pvc", v.PersistentVolumeClaim.ClaimName)
	}
	if v.EmptyDir != nil {
		t.Error("expected no EmptyDir volume source for a non-ephemeral volume")
	}
}

func TestBuildAgentVolumes_Ephemeral(t *testing.T) {
	vols := buildAgentVolumes("workspace", "my-agent-pvc", true, nil)
	if len(vols) != 1 {
		t.Fatalf("expected 1 volume, got %d", len(vols))
	}
	v := vols[0]
	if v.EmptyDir == nil {
		t.Fatal("expected an EmptyDir volume source for an ephemeral volume")
	}
	if v.PersistentVolumeClaim != nil {
		t.Error("expected no PersistentVolumeClaim volume source for an ephemeral volume")
	}
	if v.EmptyDir.SizeLimit != nil {
		t.Error("expected no SizeLimit when none was requested")
	}
}

func TestBuildAgentVolumes_EphemeralWithSizeLimit(t *testing.T) {
	limit := resource.MustParse("10Gi")
	vols := buildAgentVolumes("workspace", "my-agent-pvc", true, &limit)
	v := vols[0]
	if v.EmptyDir == nil || v.EmptyDir.SizeLimit == nil {
		t.Fatal("expected an EmptyDir volume source with a SizeLimit")
	}
	if v.EmptyDir.SizeLimit.String() != "10Gi" {
		t.Errorf("sizeLimit = %s, want 10Gi", v.EmptyDir.SizeLimit.String())
	}
}
