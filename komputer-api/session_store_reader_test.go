package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

var sessionFixture = []byte(strings.Join([]string{
	`{"timestamp":"2024-01-01T00:00:00Z","message":{"role":"user","content":"do the thing"}}`,
	`{"timestamp":"2024-01-01T00:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"done"}],"usage":{"input_tokens":10,"output_tokens":5},"model":"claude-sonnet-4-6"}}`,
}, "\n"))

func sessionTarball(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type fakeGetter struct {
	body   []byte
	bodies map[string][]byte // per key, when set
	err   error
	calls atomic.Int32
	delay time.Duration
	keys  []string
	mu    sync.Mutex
}

func (f *fakeGetter) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	f.calls.Add(1)
	f.mu.Lock()
	f.keys = append(f.keys, bucket+"/"+key)
	f.mu.Unlock()
	time.Sleep(f.delay)
	if f.err != nil {
		return nil, f.err
	}
	if f.bodies != nil {
		return io.NopCloser(bytes.NewReader(f.bodies[key])), nil
	}
	return io.NopCloser(bytes.NewReader(f.body)), nil
}

func storeBackfillAgent(podName string) *komputerv1alpha1.KomputerAgent {
	return &komputerv1alpha1.KomputerAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "a1", Namespace: "default", UID: "u-1"},
		Spec:       komputerv1alpha1.KomputerAgentSpec{AgentConfigSpec: komputerv1alpha1.AgentConfigSpec{Model: "claude-sonnet-4-6"}},
		Status:     komputerv1alpha1.KomputerAgentStatus{Phase: komputerv1alpha1.AgentPhaseArchived, PodName: podName, SessionID: "s1"},
	}
}

func useFakeGetter(t *testing.T, g sessionObjectGetter) {
	t.Helper()
	orig := newSessionObjectGetter
	newSessionObjectGetter = func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (sessionObjectGetter, error) { return g, nil }
	t.Cleanup(func() { newSessionObjectGetter = orig })
}

func TestFetchSessionEventsFromStore_ConvertsJSONL(t *testing.T) {
	g := &fakeGetter{body: sessionTarball(t, map[string][]byte{
		".claude/projects/-workspace/s1.jsonl": sessionFixture,
		".komputer-session":                    []byte("s1"),
	})}
	s3 := &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "sessions", Prefix: "komputer"}
	got, err := fetchSessionEventsFromStore(context.Background(), g, s3, storeBackfillAgent(""), 0)
	if err != nil {
		t.Fatal(err)
	}
	want := convertSessionJSONL(sessionFixture, "a1", 0, "claude-sonnet-4-6", 0)
	if len(got) == 0 || !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	if g.keys[0] != "sessions/komputer/default/a1/u-1/latest.tar.gz" {
		t.Errorf("fetched %q", g.keys[0])
	}
}

func TestFetchSessionEventsFromStore_CorruptLatest_ReadsPrevious(t *testing.T) {
	good := sessionTarball(t, map[string][]byte{".claude/projects/-workspace/s1.jsonl": sessionFixture})
	g := &fakeGetter{bodies: map[string][]byte{
		"komputer/default/a1/u-1/latest.tar.gz":   good[:len(good)/2], // a bad save
		"komputer/default/a1/u-1/previous.tar.gz": good,
	}}
	s3 := &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "sessions", Prefix: "komputer"}
	got, err := fetchSessionEventsFromStore(context.Background(), g, s3, storeBackfillAgent(""), 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := convertSessionJSONL(sessionFixture, "a1", 0, "claude-sonnet-4-6", 0); !reflect.DeepEqual(got, want) {
		t.Fatalf("events = %+v, want the previous session's", got)
	}
}

func TestFetchSessionEventsFromStore_FetchError_DoesNotTryPrevious(t *testing.T) {
	g := &fakeGetter{err: errors.New("connection refused")}
	s3 := &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "sessions", Prefix: "komputer"}
	if _, err := fetchSessionEventsFromStore(context.Background(), g, s3, storeBackfillAgent(""), 0); err == nil {
		t.Fatal("want the fetch error")
	}
	if n := g.calls.Load(); n != 1 {
		t.Errorf("GetObject calls = %d, want 1", n)
	}
}

func TestFetchSessionEventsFromStore_MissingSessionFile_ReturnsEmpty(t *testing.T) {
	g := &fakeGetter{body: sessionTarball(t, map[string][]byte{".komputer-session": []byte("s1")})}
	got, err := fetchSessionEventsFromStore(context.Background(), g, &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b"}, storeBackfillAgent(""), 0)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want empty, nil", got, err)
	}
}

func TestSessionBackfill_NoPodS3On_UsesStore(t *testing.T) {
	g := &fakeGetter{body: sessionTarball(t, map[string][]byte{".claude/projects/-workspace/s1.jsonl": sessionFixture})}
	useFakeGetter(t, g)
	k8s := newFakeK8s(t, s3Config(true))
	var persisted []AgentEvent
	got := sessionBackfillEvents(context.Background(), k8s, "default", storeBackfillAgent(""), func(evs []AgentEvent) { persisted = evs })
	if len(got) == 0 || !reflect.DeepEqual(got, persisted) {
		t.Fatalf("events = %d, persisted = %d", len(got), len(persisted))
	}
}

func TestSessionBackfill_S3Off_NoPod_ReturnsNothing(t *testing.T) {
	g := &fakeGetter{}
	useFakeGetter(t, g)
	got := sessionBackfillEvents(context.Background(), newFakeK8s(t, s3Config(false)), "default", storeBackfillAgent(""), func([]AgentEvent) {})
	if len(got) != 0 || g.calls.Load() != 0 {
		t.Fatalf("got %d events, %d S3 calls; want none", len(got), g.calls.Load())
	}
}

func TestSessionBackfill_PodRunning_PrefersPod(t *testing.T) {
	g := &fakeGetter{}
	useFakeGetter(t, g)
	orig := podSessionEvents
	podSessionEvents = func(ctx context.Context, k8s *K8sClient, ns, podName, sessionID, agentName string, limit int64, model string, totalCostUSD float64) []AgentEvent {
		return []AgentEvent{{AgentName: agentName, Type: "text"}}
	}
	t.Cleanup(func() { podSessionEvents = orig })

	got := sessionBackfillEvents(context.Background(), newFakeK8s(t, s3Config(true)), "default", storeBackfillAgent("a1-pod"), func([]AgentEvent) {})
	if len(got) != 1 || g.calls.Load() != 0 {
		t.Fatalf("got %d events, %d S3 calls; want the pod's 1 event and no S3 call", len(got), g.calls.Load())
	}
}

func TestSessionBackfill_ConcurrentRequestsShareOneDownload(t *testing.T) {
	g := &fakeGetter{body: sessionTarball(t, map[string][]byte{".claude/projects/-workspace/s1.jsonl": sessionFixture}), delay: 50 * time.Millisecond}
	useFakeGetter(t, g)
	k8s := newFakeK8s(t, s3Config(true))
	var persists atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sessionBackfillEvents(context.Background(), k8s, "default", storeBackfillAgent(""), func([]AgentEvent) { persists.Add(1) })
		}()
	}
	wg.Wait()
	if g.calls.Load() != 1 || persists.Load() != 1 {
		t.Fatalf("S3 calls = %d, persists = %d; want 1 and 1", g.calls.Load(), persists.Load())
	}
}

func TestSessionBackfill_StoreError_ReturnsEmpty(t *testing.T) {
	useFakeGetter(t, &fakeGetter{err: errors.New("AccessDenied")})
	got := sessionBackfillEvents(context.Background(), newFakeK8s(t, s3Config(true)), "default", storeBackfillAgent(""), func([]AgentEvent) {
		t.Error("nothing should be persisted on a store error")
	})
	if len(got) != 0 {
		t.Fatalf("got %d events, want 0", len(got))
	}
}
