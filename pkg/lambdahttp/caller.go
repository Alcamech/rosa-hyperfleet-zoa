package lambdahttp

import (
	"context"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/awsarn"
)

type callerARNContextKey struct{}

type callerAccountIDContextKey struct{}

// ContextWithAuthenticatedCallerARN stores the SigV4-authenticated principal ARN
// from the Lambda Function URL IAM authorizer (requestContext.authorizer.iam.userArn).
func ContextWithAuthenticatedCallerARN(ctx context.Context, arn string) context.Context {
	return context.WithValue(ctx, callerARNContextKey{}, arn)
}

// ContextWithAuthenticatedAccountID stores the AWS account id from the IAM authorizer
// (requestContext.authorizer.iam.accountId).
func ContextWithAuthenticatedAccountID(ctx context.Context, accountID string) context.Context {
	return context.WithValue(ctx, callerAccountIDContextKey{}, accountID)
}

// AuthenticatedCallerARN returns the IAM authorizer userArn attached to the request context.
func AuthenticatedCallerARN(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	arn, _ := ctx.Value(callerARNContextKey{}).(string)
	return arn
}

// AuthenticatedAccountID returns the IAM authorizer account id from context, or parses it
// from the authenticated caller ARN when the authorizer field is empty.
func AuthenticatedAccountID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(callerAccountIDContextKey{}).(string); ok && id != "" {
		return id
	}
	return awsarn.AccountID(AuthenticatedCallerARN(ctx))
}

// HasAuthenticatedIAM reports whether both caller ARN and AWS account are present.
func HasAuthenticatedIAM(ctx context.Context) bool {
	return AuthenticatedCallerARN(ctx) != "" && AuthenticatedAccountID(ctx) != ""
}
