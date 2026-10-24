package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

func s3Config(enabled bool) *komputerv1alpha1.KomputerConfig {
	cfg := &komputerv1alpha1.KomputerConfig{ObjectMeta: metav1.ObjectMeta{Name: "default"}}
	if enabled {
		cfg.Spec.S3 = &komputerv1alpha1.S3Spec{
			Enabled: true, Bucket: "sessions", Region: "us-east-1",
			CredentialsSecret: &komputerv1alpha1.S3CredentialsSecretRef{Name: "s3-creds", Namespace: "komputer-system"},
		}
	}
	return cfg
}

func TestS3Config_ReadsFirstKomputerConfig(t *testing.T) {
	k8s := newFakeK8s(t, s3Config(true))
	got := k8s.S3Config(context.Background())
	if got == nil || got.Bucket != "sessions" {
		t.Fatalf("S3Config = %+v, want bucket sessions", got)
	}
	if !k8s.S3Enabled(context.Background()) {
		t.Error("S3Enabled = false, want true")
	}
}

func TestS3Config_NoConfig_ReturnsNil(t *testing.T) {
	k8s := newFakeK8s(t)
	if got := k8s.S3Config(context.Background()); got != nil {
		t.Errorf("S3Config = %+v, want nil", got)
	}
	if k8s.S3Enabled(context.Background()) {
		t.Error("S3Enabled = true with no KomputerConfig")
	}
}

func TestGetConfig_NeverReturnsCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/config", getConfig(newFakeK8s(t, s3Config(true))))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/config", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "credentials") || strings.Contains(w.Body.String(), "s3-creds") {
		t.Errorf("config response leaks credentials: %s", w.Body.String())
	}
	var resp ConfigResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.S3.Enabled || resp.S3.Bucket != "sessions" {
		t.Errorf("resp = %+v", resp)
	}
}

func TestExtractSessionStore(t *testing.T) {
	ev := AgentEvent{Type: "task_completed", Payload: map[string]interface{}{
		"session_store": map[string]interface{}{"ok": false, "at": "2026-10-24T10:00:00Z", "bytes": float64(12), "error": "AccessDenied"},
	}}
	got := extractSessionStore(ev)
	if got == nil || got.OK || got.Error != "AccessDenied" || got.Bytes != 12 || got.At.IsZero() {
		t.Fatalf("extractSessionStore = %+v", got)
	}
	if extractSessionStore(AgentEvent{Type: "task_completed", Payload: map[string]interface{}{}}) != nil {
		t.Error("no session_store in payload should yield nil")
	}
}

func storeAgent(lifecycle komputerv1alpha1.AgentLifecycle) *komputerv1alpha1.KomputerAgent {
	started := metav1.NewTime(time.Now().Add(-time.Minute))
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Lifecycle: lifecycle}},
		Status:     komputerv1alpha1.KomputerAgentStatus{TaskStatus: komputerv1alpha1.AgentTaskInProgress, TaskStartedAt: &started},
	}
}

func getStoredAgent(t *testing.T, k8s *K8sClient) *komputerv1alpha1.KomputerAgent {
	t.Helper()
	a := &komputerv1alpha1.KomputerAgent{}
	if err := k8s.client.Get(context.Background(), types.NamespacedName{Name: "a1", Namespace: "default"}, a); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestPatchAgentTaskEvent_RecordsSessionStore(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, storeAgent(""))
	at := time.Now()
	store := &SessionStoreResult{OK: true, At: at, Bytes: 10}
	if err := k8s.PatchAgentTaskEvent(context.Background(), "task_completed", "default", "a1", "Complete", "Task completed", "s1", 0, 0, 0, store); err != nil {
		t.Fatal(err)
	}
	got := getStoredAgent(t, k8s).Status
	if got.SessionStore == nil || !got.SessionStore.OK || got.SessionStore.LastSavedAt == nil || got.SessionStore.LastAttemptAt == nil {
		t.Fatalf("sessionStore = %+v", got.SessionStore)
	}
	if !komputerv1alpha1.LatestTaskSaved(getStoredAgent(t, k8s)) {
		t.Error("LatestTaskSaved should be true right after a successful save")
	}
}

func TestPatchAgentTaskEvent_FailedSaveOnArchiveBecomesError(t *testing.T) {
	for _, lc := range []komputerv1alpha1.AgentLifecycle{komputerv1alpha1.AgentLifecycleArchive, komputerv1alpha1.AgentLifecycleAutoDelete} {
		k8s := newFakeK8sWithStatusSubresource(t, storeAgent(lc))
		store := &SessionStoreResult{OK: false, At: time.Now(), Error: "AccessDenied: no write"}
		if err := k8s.PatchAgentTaskEvent(context.Background(), "task_completed", "default", "a1", "Complete", "Task completed", "s1", 0, 0, 0, store); err != nil {
			t.Fatal(err)
		}
		got := getStoredAgent(t, k8s).Status
		if got.TaskStatus != komputerv1alpha1.AgentTaskError {
			t.Errorf("%s: TaskStatus = %q, want Error", lc, got.TaskStatus)
		}
		if !strings.HasPrefix(got.LastTaskMessage, "Couldn't save the session to S3: ") {
			t.Errorf("%s: LastTaskMessage = %q", lc, got.LastTaskMessage)
		}
		if got.SessionStore == nil || got.SessionStore.OK || got.SessionStore.LastSavedAt != nil {
			t.Errorf("%s: sessionStore = %+v", lc, got.SessionStore)
		}
	}
}

func TestPatchAgentTaskEvent_FailedSaveOnDefaultLifecycleStaysComplete(t *testing.T) {
	k8s := newFakeK8sWithStatusSubresource(t, storeAgent(""))
	store := &SessionStoreResult{OK: false, At: time.Now(), Error: "boom"}
	if err := k8s.PatchAgentTaskEvent(context.Background(), "task_completed", "default", "a1", "Complete", "Task completed", "s1", 0, 0, 0, store); err != nil {
		t.Fatal(err)
	}
	if got := getStoredAgent(t, k8s).Status.TaskStatus; got != komputerv1alpha1.AgentTaskComplete {
		t.Errorf("TaskStatus = %q, want Complete", got)
	}
}

func TestAgentResponse_SessionStoreFromConditionOnCopyOutFailure(t *testing.T) {
	saved := metav1.NewTime(time.Now())
	agent := storeAgent("")
	agent.Status.SessionStore = &komputerv1alpha1.SessionStoreStatus{OK: true, LastSavedAt: &saved, LastAttemptAt: &saved}
	meta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
		Type: komputerv1alpha1.ConditionSessionSaved, Status: metav1.ConditionFalse, Reason: "SaveFailed", Message: "Archive stopped: couldn't save the session to S3: boom",
	})
	var resp AgentResponse
	fillAgentTTL(&resp, agent)
	if resp.SessionStore == nil || resp.SessionStore.OK || resp.SessionStore.Error != "Archive stopped: couldn't save the session to S3: boom" {
		t.Fatalf("sessionStore = %+v", resp.SessionStore)
	}
	if resp.SessionStore.LastSavedAt == "" {
		t.Error("lastSavedAt should still be reported")
	}
}
