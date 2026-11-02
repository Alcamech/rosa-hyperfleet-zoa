package accessclient

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockDeploymentSSMLister struct {
	getParametersByPathFn func(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error)
}

func (m *mockDeploymentSSMLister) GetParametersByPath(ctx context.Context, params *ssm.GetParametersByPathInput, optFns ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
	return m.getParametersByPathFn(ctx, params, optFns...)
}

func TestListDeploymentParametersFromSSM_WhenPaginated_ItShouldReturnAllPages(t *testing.T) {
	callCount := 0
	mock := &mockDeploymentSSMLister{
		getParametersByPathFn: func(_ context.Context, params *ssm.GetParametersByPathInput, _ ...func(*ssm.Options)) (*ssm.GetParametersByPathOutput, error) {
			callCount++
			if aws.ToString(params.Path) != ssmDeploymentsPath+"/" {
				t.Fatalf("unexpected path %q", aws.ToString(params.Path))
			}
			if callCount == 1 {
				return &ssm.GetParametersByPathOutput{
					Parameters: []ssmtypes.Parameter{
						{Name: aws.String("/zoa/deployments/us-east-1-eph-aaaa")},
					},
					NextToken: aws.String("page2"),
				}, nil
			}
			return &ssm.GetParametersByPathOutput{
				Parameters: []ssmtypes.Parameter{
					{Name: aws.String("/zoa/deployments/us-east-1-eph-175cbb02")},
				},
			}, nil
		},
	}

	params, err := listDeploymentParametersFromSSM(context.Background(), mock)
	if err != nil {
		t.Fatalf("listDeploymentParametersFromSSM: %v", err)
	}
	if len(params) != 2 {
		t.Fatalf("expected 2 parameters across pages, got %d", len(params))
	}
	if callCount != 2 {
		t.Fatalf("expected 2 SSM API calls, got %d", callCount)
	}
}
