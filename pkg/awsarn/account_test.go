package awsarn

import "testing"

func TestAccountID_WhenAssumedRoleARN_ItShouldReturnAccount(t *testing.T) {
	got := AccountID("arn:aws:sts::599476212575:assumed-role/zoa-access-invoker/slopezma")
	if got != "599476212575" {
		t.Fatalf("got %q", got)
	}
}

func TestAccountID_WhenIAMRoleARN_ItShouldReturnAccount(t *testing.T) {
	got := AccountID("arn:aws:iam::123456789012:role/foo-zoa-access-invoker")
	if got != "123456789012" {
		t.Fatalf("got %q", got)
	}
}

func TestAccountID_WhenInvalid_ItShouldReturnEmpty(t *testing.T) {
	if AccountID("") != "" {
		t.Fatal("expected empty")
	}
	if AccountID("not-an-arn") != "" {
		t.Fatal("expected empty")
	}
}
