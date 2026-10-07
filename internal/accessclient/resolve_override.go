package accessclient

import (
	"context"
	"fmt"
	"strings"
)

// ErrDirectAccessPairRequired is returned when only one direct-Access override flag is set.
var ErrDirectAccessPairRequired = fmt.Errorf("direct Access override: --access-url and --invoker-role-arn must be used together (or omit both for deployment SSM autodiscovery)")

// DirectAccessOverrideActive reports whether both direct Access endpoint flags are set.
func DirectAccessOverrideActive(accessURL, invokerRoleARN string) bool {
	return strings.TrimSpace(accessURL) != "" && strings.TrimSpace(invokerRoleARN) != ""
}

// ValidateDirectAccessPair returns an error if exactly one override is set.
func ValidateDirectAccessPair(accessURL, invokerRoleARN string) error {
	hasURL := strings.TrimSpace(accessURL) != ""
	hasInvoker := strings.TrimSpace(invokerRoleARN) != ""
	if hasURL != hasInvoker {
		return ErrDirectAccessPairRequired
	}
	return nil
}

// ResolveDeployment resolves Access URL and invoker credentials from Central SSM
// or from explicit --access-url / --invoker-role-arn (skips Central deployment SSM autodiscovery).
func ResolveDeployment(ctx context.Context, deploymentName, deploymentsRegion, accessURL, invokerRoleARN string) (*DeploymentInfo, error) {
	if err := ValidateDirectAccessPair(accessURL, invokerRoleARN); err != nil {
		return nil, err
	}
	if DirectAccessOverrideActive(accessURL, invokerRoleARN) {
		return resolveFromDirectAccess(ctx, strings.TrimSpace(accessURL), strings.TrimSpace(invokerRoleARN))
	}
	return Resolve(ctx, deploymentName, deploymentsRegion)
}

func resolveFromDirectAccess(ctx context.Context, accessURL, invokerRoleARN string) (*DeploymentInfo, error) {
	if accessURL == "" || invokerRoleARN == "" {
		return nil, fmt.Errorf("direct Access override: access URL and invoker role ARN are required")
	}

	cfg, err := AWSConfigForDeploymentsSSM(ctx, DefaultDeploymentsSSMRegion)
	if err != nil {
		return nil, fmt.Errorf("loading AWS config: %w", err)
	}

	sessionName, err := extractSessionName(ctx, cfg)
	if err != nil {
		return nil, err
	}

	creds := assumeInvokerRole(cfg, invokerRoleARN, sessionName)

	operatorARN, err := callerARN(ctx, cfg, creds)
	if err != nil {
		return nil, fmt.Errorf("invoker role identity: %w", err)
	}

	accountID := accountIDFromIAMARN(invokerRoleARN)

	return &DeploymentInfo{
		AccessURL:   accessURL,
		Credentials: creds,
		AccountID:   accountID,
		SessionName: sessionName,
		OperatorARN: operatorARN,
	}, nil
}

// accountIDFromIAMARN returns the 12-digit account from arn:aws:iam::ACCOUNT:...
func accountIDFromIAMARN(arn string) string {
	parts := strings.Split(arn, ":")
	if len(parts) < 5 {
		return ""
	}
	id := parts[4]
	if len(id) != 12 {
		return ""
	}
	return id
}
