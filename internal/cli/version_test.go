package cli

import (
	"context"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
)

func TestApplyAPIURLFromEnv_WhenFlagEmpty_ItShouldUseEnv(t *testing.T) {
	t.Setenv("ZOA_API_URL", "https://from-env.lambda-url.on.aws/")
	opts := &GlobalOptions{}
	applyAPIURLFromEnv(opts)
	if opts.APIURL != "https://from-env.lambda-url.on.aws/" {
		t.Fatalf("expected env URL, got %q", opts.APIURL)
	}
}

func TestApplyAPIURLFromEnv_WhenFlagSet_ItShouldKeepFlag(t *testing.T) {
	t.Setenv("ZOA_API_URL", "https://from-env.lambda-url.on.aws/")
	opts := &GlobalOptions{APIURL: "https://from-flag.lambda-url.on.aws/"}
	applyAPIURLFromEnv(opts)
	if opts.APIURL != "https://from-flag.lambda-url.on.aws/" {
		t.Fatalf("expected flag URL to win, got %q", opts.APIURL)
	}
}

func TestVersionCommand_WhenAPIURLFromEnv_ItShouldQueryServer(t *testing.T) {
	t.Setenv("ZOA_API_URL", "https://test.lambda-url.on.aws/")
	var serverCalled bool
	mock := &mockClient{
		serverVersionFn: func(_ context.Context) (*client.ServerVersionInfo, error) {
			serverCalled = true
			return &client.ServerVersionInfo{
				Version:   "0.5.0",
				GitCommit: "test",
				BuildDate: "2026-01-01T00:00:00Z",
				GoVersion: "go1.26",
				Platform:  "linux/amd64",
				Target:    "eph-test-regional",
			}, nil
		},
	}
	global := &GlobalOptions{
		ClientFactory: func(_ *GlobalOptions) (APIClient, error) {
			return mock, nil
		},
	}
	applyAPIURLFromEnv(global)
	if global.APIURL == "" {
		t.Fatal("expected APIURL from env before version runs")
	}

	cmd := newVersionCommand(global)
	if err := cmd.RunE(cmd, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !serverCalled {
		t.Fatal("expected ServerVersion when APIURL is set from env")
	}
}
