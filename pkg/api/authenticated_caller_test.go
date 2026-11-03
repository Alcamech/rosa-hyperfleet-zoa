package api

import (
	"net/http"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/awsarn"
	"github.com/openshift-online/rosa-seaplane-zoa/pkg/lambdahttp"
)

const (
	testCallerAccountID = "123456789012"
	testLaptopCallerARN = "arn:aws:sts::123456789012:assumed-role/sre-role/slopezma"
)

// attachAuthenticatedIdentityForTest mirrors production IAM authorizer fields on request context.
func attachAuthenticatedIdentityForTest(r *http.Request, callerARN, accountID string) *http.Request {
	if accountID == "" && callerARN != "" {
		accountID = awsarn.AccountID(callerARN)
	}
	ctx := r.Context()
	if callerARN != "" {
		ctx = lambdahttp.ContextWithAuthenticatedCallerARN(ctx, callerARN)
	}
	if accountID != "" {
		ctx = lambdahttp.ContextWithAuthenticatedAccountID(ctx, accountID)
	}
	return r.WithContext(ctx)
}
