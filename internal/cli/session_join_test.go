package cli

import (
	"context"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
	"github.com/openshift-online/rosa-seaplane-zoa/internal/output"
)

func TestSessionJoin_WhenExecScriptWithJSONOutput_ItShouldRejectBeforeAPI(t *testing.T) {
	mock := &mockClient{
		sessionJoinFn: func(_ context.Context, _ string) (*client.SessionJoinResponse, error) {
			t.Fatal("SessionJoin should not be called when --exec-script and -o json")
			return nil, nil
		},
	}
	opts := newMockGlobalOpts(mock)
	opts.OutputFormat = output.FormatJSON
	root := sessionTestRoot(opts)
	root.SetArgs([]string{
		"session", "join", "us-east-1/sess-join-exec-script",
		"--exec-script", "true",
	})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error for --exec-script with -o json")
	}
}
