package controller

import (
	"context"
	"fmt"
	"io"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// SessionUploader puts one session object into the S3 session store.
type SessionUploader interface {
	Upload(ctx context.Context, key string, body io.Reader) error
}

// UploaderFactory builds a SessionUploader for the configured store. Swappable
// so tests don't talk to S3.
type UploaderFactory func(ctx context.Context, s3 *komputerv1alpha1.S3Spec) (SessionUploader, error)

type s3Uploader struct {
	uploader *manager.Uploader
	bucket   string
}

func (u *s3Uploader) Upload(ctx context.Context, key string, body io.Reader) error {
	_, err := u.uploader.Upload(ctx, &s3.PutObjectInput{Bucket: &u.bucket, Key: &key, Body: body})
	return err
}

// NewS3Uploader uses the AWS default credential chain only: static keys arrive
// as AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY env vars that Helm injects into the
// operator Deployment (from s3.credentials.secretName, or operator.extraEnv), or
// an IAM role on the operator's ServiceAccount. It never reads Secrets itself.
// The streaming upload manager handles bodies of unknown length (an exec pipe).
func NewS3Uploader(ctx context.Context, spec *komputerv1alpha1.S3Spec) (SessionUploader, error) {
	region := spec.Region
	if region == "" {
		region = "us-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if spec.Endpoint != "" {
			endpoint := spec.Endpoint
			o.BaseEndpoint = &endpoint
		}
		o.UsePathStyle = spec.ForcePathStyle
	})
	return &s3Uploader{uploader: manager.NewUploader(client), bucket: spec.Bucket}, nil
}
