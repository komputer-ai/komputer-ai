package v1alpha1

import (
	"path"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3 session store constants.
const (
	// ConditionSessionSaved reports whether the operator could save an agent's
	// session to S3 on its behalf (copy-out). Owned by the operator.
	ConditionSessionSaved = "SessionSaved"
	// FinalizerSessionStore holds an agent's deletion until its latest session
	// is saved to S3. Added by the operator only while S3 is configured.
	FinalizerSessionStore = "komputer.ai/session-store"
	// LabelSessionSaver marks the short-lived pods the operator starts to save
	// the session of an agent that only has a PVC left.
	LabelSessionSaver = "komputer.ai/session-saver"
	// S3CredentialsAccessKeyKey / S3CredentialsSecretKeyKey are the keys the
	// S3 credentials Secret must carry.
	S3CredentialsAccessKeyKey = "AWS_ACCESS_KEY_ID"
	S3CredentialsSecretKeyKey = "AWS_SECRET_ACCESS_KEY"
)

// S3Spec configures the S3 session store: where agent sessions (.claude) are
// saved and restored from. Any S3-compatible store works (set Endpoint, and
// usually ForcePathStyle, for MinIO / R2 / GCS / Ceph).
type S3Spec struct {
	// Enabled turns the session store on.
	Enabled bool `json:"enabled"`
	// Bucket holds the agents' session objects.
	Bucket string `json:"bucket"`
	// Region of the bucket.
	// +optional
	Region string `json:"region,omitempty"`
	// Endpoint is the S3 API URL for non-AWS stores. Empty means AWS.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
	// ForcePathStyle uses path-style addressing (most non-AWS stores need it).
	// +optional
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`
	// Prefix is prepended to every object key.
	// +optional
	Prefix string `json:"prefix,omitempty"`
	// CredentialsSecret holds static credentials under the keys
	// AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. When unset, everything uses
	// the AWS default credential chain (IAM roles, or env vars you add).
	// +optional
	CredentialsSecret *S3CredentialsSecretRef `json:"credentialsSecret,omitempty"`
}

// S3CredentialsSecretRef points at the Secret with the S3 credentials.
type S3CredentialsSecretRef struct {
	// Name of the Secret.
	Name string `json:"name"`
	// Namespace of the Secret. Empty means the operator's namespace.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// IsEnabled reports whether the S3 session store is configured. Nil-safe.
func (s *S3Spec) IsEnabled() bool {
	return s != nil && s.Enabled && s.Bucket != ""
}

// SessionStoreKey is the object key holding an agent's session:
// <prefix>/<namespace>/<name>/<uid>/session.tar.gz. The UID keeps a new agent
// that reuses a deleted agent's name from resuming the old session.
func SessionStoreKey(s *S3Spec, agent *KomputerAgent) string {
	prefix := ""
	if s != nil {
		prefix = strings.Trim(s.Prefix, "/")
	}
	return path.Join(prefix, agent.Namespace, agent.Name, string(agent.UID), "session.tar.gz")
}

// SessionStoreStatus is the result of the agent's own session saves.
// Owned by the API worker, which records it from the task-end event.
type SessionStoreStatus struct {
	// LastSavedAt is when the session was last saved successfully.
	// +optional
	LastSavedAt *metav1.Time `json:"lastSavedAt,omitempty"`
	// LastAttemptAt is when a save was last attempted.
	// +optional
	LastAttemptAt *metav1.Time `json:"lastAttemptAt,omitempty"`
	// OK reports whether the last attempt succeeded.
	OK bool `json:"ok"`
	// Error is the last attempt's error, if it failed.
	// +optional
	Error string `json:"error,omitempty"`
}

// LatestTaskSaved reports whether the agent's latest task has a saved session
// in S3 — either the agent's own save or an operator copy-out made after the
// task started. An agent that never ran a task has nothing to save.
func LatestTaskSaved(agent *KomputerAgent) bool {
	started := agent.Status.TaskStartedAt
	if started == nil {
		return true
	}
	if s := agent.Status.SessionStore; s != nil && s.OK && s.LastSavedAt != nil && !s.LastSavedAt.Before(started) {
		return true
	}
	if c := agent.Status.SessionCopiedOutAt; c != nil && !c.Before(started) {
		return true
	}
	return false
}
