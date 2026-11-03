package api

import (
	"net/http"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/lambdahttp"
)

// authenticatedCallerARN returns the SigV4 caller ARN from the Lambda IAM authorizer context.
func authenticatedCallerARN(r *http.Request) string {
	return lambdahttp.AuthenticatedCallerARN(r.Context())
}

// authenticatedAccountID returns the caller AWS account from the IAM authorizer (not client headers).
func authenticatedAccountID(r *http.Request) string {
	return lambdahttp.AuthenticatedAccountID(r.Context())
}

// ensureAuthenticatedIAM rejects requests that reached the handler without Lambda IAM authorizer identity.
func ensureAuthenticatedIAM(w http.ResponseWriter, r *http.Request) bool {
	if lambdahttp.HasAuthenticatedIAM(r.Context()) {
		return true
	}
	writeError(w, http.StatusUnauthorized, "unauthenticated", "SigV4 IAM caller identity required")
	return false
}
