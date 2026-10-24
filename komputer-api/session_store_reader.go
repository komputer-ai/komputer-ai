// session_store_reader.go — chat history from the S3 session store.
//
// When Redis has no history for an agent and the agent has no pod (sleeping or
// archived), the pod backfill in backfill.go can't run. If the S3 session store
// is on, the agent's last saved session is in the bucket: read the transcript out
// of the tarball and convert it with the same converter the pod path uses.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/sync/singleflight"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// sessionObjectGetter fetches one object from the session store.
type sessionObjectGetter interface {
	GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error)
}

// s3Getter is the aws-sdk-go-v2 sessionObjectGetter (see loadS3Config for
// credentials).
type s3Getter struct{ client *s3.Client }

func (g *s3Getter) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	out, err := g.client.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

// loadS3Config builds the AWS config for the session store. Static keys arrive
// as KOMPUTER_S3_ACCESS_KEY_ID/KOMPUTER_S3_SECRET_ACCESS_KEY, injected by Helm
// from s3.credentials.secretName (or added via api.extraEnv) — named apart from
// AWS_* so they never clobber the API's other AWS settings (Bedrock). Without
// them, the AWS default credential chain applies (an IAM role on the API's
// ServiceAccount, or AWS_* env vars).
func loadS3Config(ctx context.Context, spec *komputerv1alpha1.S3Spec) (aws.Config, error) {
	region := spec.Region
	if region == "" {
		region = "us-east-1"
	}
	opts := []func(*awsconfig.LoadOptions) error{awsconfig.WithRegion(region)}
	if ak, sk := os.Getenv("KOMPUTER_S3_ACCESS_KEY_ID"), os.Getenv("KOMPUTER_S3_SECRET_ACCESS_KEY"); ak != "" && sk != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(ak, sk, "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load AWS config: %w", err)
	}
	return cfg, nil
}

func newS3Getter(ctx context.Context, spec *komputerv1alpha1.S3Spec) (sessionObjectGetter, error) {
	cfg, err := loadS3Config(ctx, spec)
	if err != nil {
		return nil, err
	}
	return &s3Getter{client: s3.NewFromConfig(cfg, func(o *s3.Options) {
		if spec.Endpoint != "" {
			endpoint := spec.Endpoint
			o.BaseEndpoint = &endpoint
		}
		o.UsePathStyle = spec.ForcePathStyle
	})}, nil
}

// Swappable in tests.
var (
	newSessionObjectGetter = newS3Getter
	podSessionEvents       = fetchSessionEvents
)

// storeBackfills makes concurrent history requests for the same agent share one
// download and one Redis write-back.
var storeBackfills singleflight.Group

// errCorruptSession marks a session object that downloaded but can't be read.
var errCorruptSession = errors.New("corrupt session tarball")

// fetchSessionEventsFromStore downloads the agent's latest session tarball and
// converts the transcript of its current session (status.sessionId). A corrupt
// latest falls back to the previous save; a failed download doesn't. A tarball
// without that transcript yields no events and no error.
func fetchSessionEventsFromStore(ctx context.Context, g sessionObjectGetter, spec *komputerv1alpha1.S3Spec, agent *komputerv1alpha1.KomputerAgent, totalCostUSD float64) ([]AgentEvent, error) {
	events, err := readSessionEvents(ctx, g, spec.Bucket, komputerv1alpha1.SessionStoreKey(spec, agent), agent, totalCostUSD)
	if errors.Is(err, errCorruptSession) {
		Logger.Warnw("latest session in S3 is unreadable; reading the previous one", "agent_name", agent.Name, "error", err)
		return readSessionEvents(ctx, g, spec.Bucket, komputerv1alpha1.SessionStorePreviousKey(spec, agent), agent, totalCostUSD)
	}
	return events, err
}

func readSessionEvents(ctx context.Context, g sessionObjectGetter, bucket, key string, agent *komputerv1alpha1.KomputerAgent, totalCostUSD float64) ([]AgentEvent, error) {
	body, err := g.GetObject(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	// Download fully before parsing, so a dropped connection stays a download
	// error and only an unreadable object counts as corrupt.
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return nil, fmt.Errorf("download session tarball: %w", err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open session tarball: %w: %w", errCorruptSession, err)
	}
	want := fmt.Sprintf(".claude/projects/-workspace/%s.jsonl", agent.Status.SessionID)
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read session tarball: %w: %w", errCorruptSession, err)
		}
		if hdr.Name != want {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read session transcript: %w: %w", errCorruptSession, err)
		}
		return convertSessionJSONL(raw, agent.Name, 0, agent.Spec.Model, totalCostUSD), nil
	}
}

// sessionBackfillEvents rebuilds an agent's chat history when Redis has none:
// from its pod when it has one (unchanged), otherwise — with the S3 session
// store on — from its saved session in S3. persist writes the events back to
// Redis; for the S3 path it runs once per burst of concurrent requests. Any S3
// failure just means no history: it never fails the request.
func sessionBackfillEvents(ctx context.Context, k8s *K8sClient, ns string, agent *komputerv1alpha1.KomputerAgent, persist func([]AgentEvent)) []AgentEvent {
	if agent.Status.SessionID == "" {
		return nil
	}
	var totalCost float64
	if agent.Status.TotalCostUSD != "" {
		fmt.Sscanf(agent.Status.TotalCostUSD, "%f", &totalCost)
	}

	if agent.Status.PodName != "" {
		events := podSessionEvents(ctx, k8s, ns, agent.Status.PodName, agent.Status.SessionID, agent.Name, 0, agent.Spec.Model, totalCost)
		if len(events) > 0 {
			persist(events)
		}
		return events
	}

	spec := k8s.S3Config(ctx)
	if !spec.IsEnabled() {
		return nil
	}
	v, _, _ := storeBackfills.Do(ns+"/"+agent.Name, func() (interface{}, error) {
		g, err := newSessionObjectGetter(ctx, spec)
		if err != nil {
			Logger.Warnw("session store backfill: no S3 client", "agent_name", agent.Name, "error", err)
			return []AgentEvent(nil), nil
		}
		events, err := fetchSessionEventsFromStore(ctx, g, spec, agent, totalCost)
		if err != nil {
			Logger.Warnw("session store backfill failed", "agent_name", agent.Name, "error", err)
			return []AgentEvent(nil), nil
		}
		Logger.Infow("chat history rebuilt from the S3 session store",
			"namespace", ns, "agent_name", agent.Name, "bucket", spec.Bucket, "event_count", len(events))
		if len(events) > 0 {
			persist(events)
		}
		return events, nil
	})
	events, _ := v.([]AgentEvent)
	return events
}
