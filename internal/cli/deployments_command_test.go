package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestDeploymentsCommand_WhenDirectAccessFlagsSet_ItShouldReject(t *testing.T) {
	root := NewRootCommand()
	var stderr bytes.Buffer
	root.SetOut(bytes.NewBuffer(nil))
	root.SetErr(&stderr)
	root.SetArgs([]string{
		"deployments",
		"--access-url", "https://example.lambda-url.us-east-1.on.aws/",
		"--invoker-role-arn", "arn:aws:iam::123456789012:role/us-east-1-zoa-access-invoker",
	})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "does not support --access-url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeploymentsCommand_WhenOnlyAccessURL_ItShouldRejectPair(t *testing.T) {
	root := NewRootCommand()
	root.SetOut(bytes.NewBuffer(nil))
	root.SetErr(bytes.NewBuffer(nil))
	root.SetArgs([]string{"deployments", "--access-url", "https://example.lambda-url.us-east-1.on.aws/"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invoker-role-arn") {
		t.Fatalf("unexpected error: %v", err)
	}
}
