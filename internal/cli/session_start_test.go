package cli

import (
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/cli/reason"
	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
)

func TestSessionStart_WhenReasonMissing_ItShouldNotCallAPI(t *testing.T) {
	mock := &mockClient{
		sessionStartFn: func(_ context.Context, _ *client.SessionStartRequest) (*client.SessionStartResponse, error) {
			t.Fatal("SessionStart should not be called without --reason")
			return nil, nil
		},
	}
	root := sessionTestRoot(newMockGlobalOpts(mock))
	root.SetArgs([]string{"session", "start", "us-east-1", "mc01", "--no-connect"})

	if err := root.Execute(); err == nil {
		t.Fatal("expected error when --reason is missing")
	}
}

func TestSessionStart_WhenReasonSet_ItShouldPassToAPI(t *testing.T) {
	var gotReason string
	mock := &mockClient{
		sessionStartFn: func(_ context.Context, req *client.SessionStartRequest) (*client.SessionStartResponse, error) {
			gotReason = req.Reason
			return &client.SessionStartResponse{
				SessionID: "sess-abc",
				Status:    "active",
				Target:    "mc01",
			}, nil
		},
	}
	root := sessionTestRoot(newMockGlobalOpts(mock))
	root.SetArgs([]string{"session", "start", "us-east-1", "mc01", "--reason", "ROSAENG-4242", "--no-connect"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotReason != "ROSAENG-4242" {
		t.Fatalf("expected reason on request, got %q", gotReason)
	}
}

func TestSessionStart_WhenNoConnect_ItShouldNotCallSessionJoin(t *testing.T) {
	mock := &mockClient{
		sessionStartFn: func(_ context.Context, _ *client.SessionStartRequest) (*client.SessionStartResponse, error) {
			return &client.SessionStartResponse{
				SessionID: "sess-nc",
				Status:    "active",
				Target:    "mc01",
			}, nil
		},
		sessionJoinFn: func(_ context.Context, _ string) (*client.SessionJoinResponse, error) {
			t.Fatal("SessionJoin should not be called with --no-connect")
			return nil, nil
		},
	}
	root := sessionTestRoot(newMockGlobalOpts(mock))
	root.SetArgs([]string{"session", "start", "us-east-1", "mc01", "--reason", "ROSAENG-1", "--no-connect"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestSessionStart_WhenConnect_ItShouldCallSessionJoin(t *testing.T) {
	var joinCalled bool
	mock := &mockClient{
		sessionStartFn: func(_ context.Context, _ *client.SessionStartRequest) (*client.SessionStartResponse, error) {
			return &client.SessionStartResponse{
				SessionID: "sess-join",
				Status:    "active",
				Target:    "mc01",
				Deadline:  "2030-01-01T00:00:00Z",
			}, nil
		},
		sessionJoinFn: func(_ context.Context, sessionID string) (*client.SessionJoinResponse, error) {
			joinCalled = true
			if sessionID != "sess-join" {
				t.Fatalf("expected session id sess-join, got %q", sessionID)
			}
			return &client.SessionJoinResponse{
				TaskArn:       "arn:aws:ecs:us-east-1:123:task/c/t",
				ClusterArn:    "arn:aws:ecs:us-east-1:123:cluster/c",
				Region:        "us-east-1",
				ContainerName: "zoa-boundary",
				ExecCommand:   "/bin/bash",
				ExecCredentials: &client.ExecCredentials{
					AccessKeyID:     "AKIA",
					SecretAccessKey: "secret",
					SessionToken:    "token",
				},
			}, nil
		},
	}
	root := sessionTestRoot(newMockGlobalOpts(mock))
	root.SetArgs([]string{"session", "start", "us-east-1", "mc01", "--reason", "ROSAENG-2"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected ECS Exec to fail in unit test environment")
	}
	if !joinCalled {
		t.Fatal("expected SessionJoin before ECS Exec")
	}
}

func TestSessionStart_WhenReasonOnlyInEnv_ItShouldPassToAPI(t *testing.T) {
	t.Setenv(reason.EnvVar, "ROSAENG-777")

	var gotReason string
	mock := &mockClient{
		sessionStartFn: func(_ context.Context, req *client.SessionStartRequest) (*client.SessionStartResponse, error) {
			gotReason = req.Reason
			return &client.SessionStartResponse{SessionID: "sess-env", Status: "active"}, nil
		},
	}
	root := sessionTestRoot(newMockGlobalOpts(mock))
	root.SetArgs([]string{"session", "start", "us-east-1", "mc01", "--no-connect"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotReason != "ROSAENG-777" {
		t.Fatalf("expected ZOA_REASON on request, got %q", gotReason)
	}
}

func sessionTestRoot(opts *GlobalOptions) *cobra.Command {
	root := &cobra.Command{Use: "zoa"}
	root.AddCommand(newSessionCommand(opts))
	return root
}
