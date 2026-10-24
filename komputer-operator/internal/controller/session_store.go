package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// SessionUploader puts one session object into the S3 session store.
type SessionUploader interface {
	Upload(ctx context.Context, key string, body io.Reader) error
	// Copy copies src to dst inside the bucket (server side). A missing src is
	// not an error: there's nothing to keep yet.
	Copy(ctx context.Context, src, dst string) error
}

// UploaderFactory builds a SessionUploader for the configured store. Swappable
// so tests don't talk to S3.
type UploaderFactory func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (SessionUploader, error)

type s3Uploader struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
}

func (u *s3Uploader) Upload(ctx context.Context, key string, body io.Reader) error {
	_, err := u.uploader.Upload(ctx, &s3.PutObjectInput{Bucket: &u.bucket, Key: &key, Body: body})
	return err
}

func (u *s3Uploader) Copy(ctx context.Context, src, dst string) error {
	segments := strings.Split(u.bucket+"/"+src, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	source := strings.Join(segments, "/")
	_, err := u.client.CopyObject(ctx, &s3.CopyObjectInput{Bucket: &u.bucket, Key: &dst, CopySource: &source})
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) && (apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound") {
		return nil
	}
	return err
}

// loadS3Config builds the AWS config for the session store. Static keys arrive
// as KOMPUTER_S3_ACCESS_KEY_ID/KOMPUTER_S3_SECRET_ACCESS_KEY, injected by Helm
// from s3.credentials.secretName (or added via operator.extraEnv) — named apart
// from AWS_* so they never clobber the operator's other AWS settings. Without
// them, the AWS default credential chain applies (an IAM role on the operator's
// ServiceAccount, or AWS_* env vars). The operator never reads Secrets itself.
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

// NewS3Uploader builds the session store client (see loadS3Config for
// credentials). The streaming upload manager handles bodies of unknown length
// (an exec pipe).
func NewS3Uploader(ctx context.Context, spec *komputerv1alpha1.S3Spec) (SessionUploader, error) {
	cfg, err := loadS3Config(ctx, spec)
	if err != nil {
		return nil, err
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if spec.Endpoint != "" {
			endpoint := spec.Endpoint
			o.BaseEndpoint = &endpoint
		}
		o.UsePathStyle = spec.ForcePathStyle
	})
	return &s3Uploader{client: client, uploader: manager.NewUploader(client), bucket: spec.Bucket}, nil
}
