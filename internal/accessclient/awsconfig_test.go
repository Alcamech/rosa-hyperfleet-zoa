package accessclient

import (
	"context"
	"testing"
)

func TestAWSConfigForDeploymentsSSM_ItShouldPinRegion(t *testing.T) {
	cfg, err := AWSConfigForDeploymentsSSM(context.Background(), "eu-west-1")
	if err != nil {
		t.Fatalf("AWSConfigForDeploymentsSSM: %v", err)
	}
	if cfg.Region != "eu-west-1" {
		t.Fatalf("expected region eu-west-1, got %q", cfg.Region)
	}
}
