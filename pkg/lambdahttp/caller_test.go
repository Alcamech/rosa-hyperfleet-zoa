package lambdahttp

import (
	"context"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/awsarn"
)

func TestAuthenticatedAccountID_WhenContextHasAuthorizerAccount_ItShouldPreferIt(t *testing.T) {
	ctx := ContextWithAuthenticatedAccountID(context.Background(), "111122223333")
	ctx = ContextWithAuthenticatedCallerARN(ctx, "arn:aws:sts::599476212575:assumed-role/x/y")
	if got := AuthenticatedAccountID(ctx); got != "111122223333" {
		t.Fatalf("got %q", got)
	}
}

func TestAuthenticatedAccountID_WhenOnlyARN_ItShouldParseAccount(t *testing.T) {
	ctx := ContextWithAuthenticatedCallerARN(context.Background(),
		"arn:aws:sts::123456789012:assumed-role/sre/slopezma")
	if got := AuthenticatedAccountID(ctx); got != "123456789012" {
		t.Fatalf("got %q", got)
	}
}

func TestHasAuthenticatedIAM_WhenComplete_ItShouldReturnTrue(t *testing.T) {
	ctx := ContextWithAuthenticatedCallerARN(context.Background(),
		"arn:aws:sts::123456789012:assumed-role/sre/slopezma")
	if !HasAuthenticatedIAM(ctx) {
		t.Fatal("expected true")
	}
}

func TestHasAuthenticatedIAM_WhenMissingARN_ItShouldReturnFalse(t *testing.T) {
	ctx := ContextWithAuthenticatedAccountID(context.Background(), "123456789012")
	if HasAuthenticatedIAM(ctx) {
		t.Fatal("expected false")
	}
}

func TestAccountIDFromARN_ReexportedViaAwsarn(t *testing.T) {
	if awsarn.AccountID("arn:aws:sts::123456789012:assumed-role/a/b") != "123456789012" {
		t.Fatal("awsarn mismatch")
	}
}
