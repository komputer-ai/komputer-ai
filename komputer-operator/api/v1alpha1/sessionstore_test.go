package v1alpha1

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestSessionStoreKey(t *testing.T) {
	agent := &KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "team-a", UID: types.UID("u-1")}}
	for _, tc := range []struct {
		prefix string
		want   string
	}{
		{"komputer", "komputer/team-a/a1/u-1/latest.tar.gz"},
		{"/komputer/", "komputer/team-a/a1/u-1/latest.tar.gz"},
		{"", "team-a/a1/u-1/latest.tar.gz"},
	} {
		if got := SessionStoreKey(&S3Spec{Prefix: tc.prefix}, agent); got != tc.want {
			t.Errorf("prefix %q: got %q, want %q", tc.prefix, got, tc.want)
		}
	}
}

func TestSessionStorePreviousKey(t *testing.T) {
	agent := &KomputerAgent{ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "team-a", UID: types.UID("u-1")}}
	if got := SessionStorePreviousKey(&S3Spec{Prefix: "komputer"}, agent); got != "komputer/team-a/a1/u-1/previous.tar.gz" {
		t.Errorf("got %q", got)
	}
}

func TestS3SpecIsEnabled(t *testing.T) {
	var nilSpec *S3Spec
	if nilSpec.IsEnabled() {
		t.Error("nil spec must be disabled")
	}
	if (&S3Spec{Enabled: true}).IsEnabled() {
		t.Error("enabled without a bucket must be disabled")
	}
	if !(&S3Spec{Enabled: true, Bucket: "b"}).IsEnabled() {
		t.Error("enabled with a bucket must be enabled")
	}
}

func TestLatestTaskSaved(t *testing.T) {
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	after := metav1.NewTime(started.Add(time.Second))
	before := metav1.NewTime(started.Add(-time.Second))

	for _, tc := range []struct {
		name   string
		status KomputerAgentStatus
		want   bool
	}{
		{"no task ever ran", KomputerAgentStatus{}, true},
		{"agent saved after the task started", KomputerAgentStatus{TaskStartedAt: &started, SessionStore: &SessionStoreStatus{OK: true, LastSavedAt: &after}}, true},
		{"agent saved before the task started", KomputerAgentStatus{TaskStartedAt: &started, SessionStore: &SessionStoreStatus{OK: true, LastSavedAt: &before}}, false},
		{"last save failed", KomputerAgentStatus{TaskStartedAt: &started, SessionStore: &SessionStoreStatus{OK: false, LastSavedAt: &after}}, false},
		{"operator copied out after the task started", KomputerAgentStatus{TaskStartedAt: &started, SessionCopiedOutAt: &after}, true},
		{"nothing saved", KomputerAgentStatus{TaskStartedAt: &started}, false},
	} {
		agent := &KomputerAgent{Status: tc.status}
		if got := LatestTaskSaved(agent); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
