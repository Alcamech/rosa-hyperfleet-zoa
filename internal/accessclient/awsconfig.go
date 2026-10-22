package accessclient

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
)

// AWSConfigForDeploymentsSSM loads credentials from the default chain and pins the
// AWS region for Central /zoa/deployments SSM reads.
func AWSConfigForDeploymentsSSM(ctx context.Context, region string) (aws.Config, error) {
	if strings.TrimSpace(region) == "" {
		region = DefaultDeploymentsSSMRegion
	}
	return awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
}
