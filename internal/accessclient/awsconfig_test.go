package accessclient

import (
	"context"
	"testing"
)

func TestAWSConfigForDeploymentsSSM_ItShouldPinRegion(t *testing.T) {
	// Do not depend on the developer's ~/.aws or AWS_PROFILE (e.g. rrp-central).
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_ACCESS_KEY_ID", "testing")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "testing")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	cfg, err := AWSConfigForDeploymentsSSM(context.Background(), "eu-west-1")
	if err != nil {
		t.Fatalf("AWSConfigForDeploymentsSSM: %v", err)
	}
	if cfg.Region != "eu-west-1" {
		t.Fatalf("expected region eu-west-1, got %q", cfg.Region)
	}
}
