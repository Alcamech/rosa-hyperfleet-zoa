package accessclient

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

func TestListDeploymentParameters_WhenPrimaryFails_ItShouldFailoverToReplica(t *testing.T) {
	restore := swapListDeploymentParametersInRegion(t)
	defer restore()

	var regions []string
	listDeploymentParametersInRegionFn = func(_ context.Context, region string) ([]ssmtypes.Parameter, error) {
		regions = append(regions, region)
		if region == DefaultDeploymentsSSMRegion {
			return nil, fmt.Errorf("SSM %s in %s: unavailable", ssmDeploymentsPath, region)
		}
		return []ssmtypes.Parameter{
			{Name: aws.String("/zoa/deployments/us-east-1"), Value: aws.String(`{"deployment_name":"us-east-1"}`)},
		}, nil
	}

	params, err := ListDeploymentParameters(context.Background(), "")
	if err != nil {
		t.Fatalf("ListDeploymentParameters: %v", err)
	}
	if len(params) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(params))
	}
	if len(regions) != 2 || regions[0] != DefaultDeploymentsSSMRegion || regions[1] != ReplicaDeploymentsSSMRegion {
		t.Fatalf("expected primary then replica, got %v", regions)
	}
}

func TestListDeploymentParameters_WhenPinnedRegion_ItShouldNotFailover(t *testing.T) {
	restore := swapListDeploymentParametersInRegion(t)
	defer restore()

	var regions []string
	listDeploymentParametersInRegionFn = func(_ context.Context, region string) ([]ssmtypes.Parameter, error) {
		regions = append(regions, region)
		return nil, errors.New("ssm error")
	}

	_, err := ListDeploymentParameters(context.Background(), "eu-west-1")
	if err == nil {
		t.Fatal("expected error")
	}
	if len(regions) != 1 || regions[0] != "eu-west-1" {
		t.Fatalf("expected single region eu-west-1, got %v", regions)
	}
}

func TestReadDeploymentParameter_WhenPrimaryFails_ItShouldFailoverToReplica(t *testing.T) {
	restore := swapReadDeploymentSSMInRegion(t)
	defer restore()

	var regions []string
	readDeploymentSSMInRegionFn = func(_ context.Context, region, name string) (*ssmDeployment, error) {
		regions = append(regions, region)
		if region == DefaultDeploymentsSSMRegion {
			return nil, fmt.Errorf("reading SSM %s/%s: unavailable", ssmDeploymentsPath, name)
		}
		return &ssmDeployment{
			DeploymentName: name,
			AccessURL:      "https://example.lambda-url.eu-west-1.on.aws/",
			InvokerRoleARN: "arn:aws:iam::123456789012:role/invoker",
			Region:         "us-east-1",
			AccountID:      "123456789012",
		}, nil
	}

	dep, err := ReadDeploymentParameter(context.Background(), "us-east-1", "")
	if err != nil {
		t.Fatalf("ReadDeploymentParameter: %v", err)
	}
	if dep.AccessURL == "" {
		t.Fatal("expected access URL")
	}
	if len(regions) != 2 {
		t.Fatalf("expected two regions tried, got %v", regions)
	}
}

func swapListDeploymentParametersInRegion(t *testing.T) func() {
	t.Helper()
	prev := listDeploymentParametersInRegionFn
	return func() { listDeploymentParametersInRegionFn = prev }
}

func swapReadDeploymentSSMInRegion(t *testing.T) func() {
	t.Helper()
	prev := readDeploymentSSMInRegionFn
	return func() { readDeploymentSSMInRegionFn = prev }
}
