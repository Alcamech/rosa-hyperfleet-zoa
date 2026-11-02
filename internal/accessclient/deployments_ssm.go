package accessclient

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// DefaultDeploymentsSSMRegion is the primary Central account region for /zoa/deployments/*.
// Must match Seaplane zoa_central_deployments_primary_region.
const DefaultDeploymentsSSMRegion = "us-east-1"

// ReplicaDeploymentsSSMRegion is the secondary replica region for deployment SSM.
// Must match Seaplane zoa_central_deployments_replica_region.
const ReplicaDeploymentsSSMRegion = "eu-west-1"

// DeploymentsSSMRegionsToTry returns Central SSM regions for /zoa/deployments reads.
// Empty deploymentsRegion enables primary then replica failover; non-empty uses only that region.
func DeploymentsSSMRegionsToTry(deploymentsRegion string) []string {
	explicit := strings.TrimSpace(deploymentsRegion)
	if explicit != "" {
		return []string{explicit}
	}
	return []string{DefaultDeploymentsSSMRegion, ReplicaDeploymentsSSMRegion}
}

// ListDeploymentParameters reads /zoa/deployments/* from Central SSM.
func ListDeploymentParameters(ctx context.Context, deploymentsRegion string) ([]ssmtypes.Parameter, error) {
	var lastErr error
	for _, region := range DeploymentsSSMRegionsToTry(deploymentsRegion) {
		params, err := listDeploymentParametersInRegionFn(ctx, region)
		if err == nil {
			return params, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no deployments SSM regions configured")
	}
	return nil, fmt.Errorf("reading Central /zoa/deployments: %w", lastErr)
}

// listDeploymentParametersInRegionFn is swappable in unit tests.
var listDeploymentParametersInRegionFn = listDeploymentParametersInRegionLive

type deploymentSSMLister interface {
	GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

func listDeploymentParametersFromSSM(ctx context.Context, client deploymentSSMLister) ([]ssmtypes.Parameter, error) {
	input := &ssm.GetParametersByPathInput{
		Path:      aws.String(ssmDeploymentsPath + "/"),
		Recursive: aws.Bool(false),
	}
	var params []ssmtypes.Parameter
	for {
		out, err := client.GetParametersByPath(ctx, input)
		if err != nil {
			return nil, err
		}
		params = append(params, out.Parameters...)
		if out.NextToken == nil {
			break
		}
		input.NextToken = out.NextToken
	}
	return params, nil
}

func listDeploymentParametersInRegionLive(ctx context.Context, region string) ([]ssmtypes.Parameter, error) {
	cfg, err := AWSConfigForDeploymentsSSM(ctx, region)
	if err != nil {
		return nil, err
	}
	params, err := listDeploymentParametersFromSSM(ctx, ssm.NewFromConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("SSM %s in %s: %w", ssmDeploymentsPath, region, err)
	}
	return params, nil
}

// ReadDeploymentParameter fetches one deployment JSON from Central SSM (with regional failover).
func ReadDeploymentParameter(ctx context.Context, deploymentName, deploymentsRegion string) (*ssmDeployment, error) {
	var lastErr error
	for _, region := range DeploymentsSSMRegionsToTry(deploymentsRegion) {
		dep, err := readDeploymentSSMInRegionFn(ctx, region, deploymentName)
		if err == nil {
			return dep, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// readDeploymentSSMInRegionFn is swappable in unit tests.
var readDeploymentSSMInRegionFn = readDeploymentSSMInRegionLive

func readDeploymentSSMInRegionLive(ctx context.Context, region, name string) (*ssmDeployment, error) {
	cfg, err := AWSConfigForDeploymentsSSM(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}
	return readDeploymentSSM(ctx, cfg, name)
}
