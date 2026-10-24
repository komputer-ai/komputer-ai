package main

import (
	"context"
	"testing"

	komputerv1alpha1 "github.com/komputer-ai/komputer-operator/api/v1alpha1"
)

// The store's static credentials are KOMPUTER_S3_* so they never clobber the
// API's own AWS settings (e.g. Bedrock via IRSA), and take precedence for S3.
func TestLoadS3Config_UsesStoreCredentialsFromEnv(t *testing.T) {
	t.Setenv("KOMPUTER_S3_ACCESS_KEY_ID", "store-ak")
	t.Setenv("KOMPUTER_S3_SECRET_ACCESS_KEY", "store-sk")
	t.Setenv("AWS_ACCESS_KEY_ID", "other-ak")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "other-sk")
	cfg, err := loadS3Config(context.Background(), &komputerv1alpha1.S3Spec{Enabled: true, Bucket: "b", Region: "eu-west-1"})
	if err != nil {
		t.Fatal(err)
	}
	creds, err := cfg.Credentials.Retrieve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if creds.AccessKeyID != "store-ak" || cfg.Region != "eu-west-1" {
		t.Errorf("access key = %q, region = %q", creds.AccessKeyID, cfg.Region)
	}
}
