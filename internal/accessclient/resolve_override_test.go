package accessclient

import (
	"context"
	"errors"
	"testing"
)

func TestDirectAccessOverrideActive_WhenBothSet_ItShouldBeTrue(t *testing.T) {
	if !DirectAccessOverrideActive("https://u", "arn:aws:iam::123456789012:role/x") {
		t.Fatal("expected active")
	}
}

func TestDirectAccessOverrideActive_WhenOnlyOneSet_ItShouldBeFalse(t *testing.T) {
	if DirectAccessOverrideActive("https://u", "") {
		t.Fatal("expected inactive with URL only")
	}
	if DirectAccessOverrideActive("", "arn:aws:iam::123456789012:role/x") {
		t.Fatal("expected inactive with ARN only")
	}
}

func TestValidateDirectAccessPair_WhenOnlyInvokerARN_ItShouldError(t *testing.T) {
	err := ValidateDirectAccessPair("", "arn:aws:iam::123456789012:role/invoker")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrDirectAccessPairRequired) {
		t.Fatalf("expected ErrDirectAccessPairRequired, got %v", err)
	}
}

func TestResolveDeployment_WhenDirectAccessPairIncomplete_ItShouldError(t *testing.T) {
	_, err := ResolveDeployment(context.Background(), "us-east-1", "", "https://example.on.aws/", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrDirectAccessPairRequired) {
		t.Fatalf("expected ErrDirectAccessPairRequired, got %v", err)
	}
}

func TestValidateDirectAccessPair_WhenOnlyAccessURL_ItShouldError(t *testing.T) {
	err := ValidateDirectAccessPair("https://example.lambda-url.us-east-1.on.aws/", "")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrDirectAccessPairRequired) {
		t.Fatalf("expected ErrDirectAccessPairRequired, got %v", err)
	}
}

func TestValidateDirectAccessPair_WhenBothOrNeither_ItShouldSucceed(t *testing.T) {
	if err := ValidateDirectAccessPair("", ""); err != nil {
		t.Fatalf("neither: %v", err)
	}
	if err := ValidateDirectAccessPair("https://u", "arn:aws:iam::123456789012:role/invoker"); err != nil {
		t.Fatalf("both: %v", err)
	}
}

func TestAccountIDFromIAMARN_WhenValidRole_ItShouldReturnAccount(t *testing.T) {
	got := accountIDFromIAMARN("arn:aws:iam::123456789012:role/foo-zoa-access-invoker")
	if got != "123456789012" {
		t.Fatalf("got %q", got)
	}
}
