package ecsexec

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
)

// JoinParams identifies a boundary task to connect to.
type JoinParams struct {
	Region        string
	Cluster       string // cluster name or ARN
	TaskARN       string
	ContainerName string
	Command       string
	NoWait        bool
	// OnExecSession runs after ecs:ExecuteCommand succeeds and before session-manager-plugin.
	// Best-effort only (errors do not block the shell); log warnings inside the callback.
	OnExecSession func(execSessionID string) error
}

// defaultExecCommand is only used when the caller omits Command (tests). Production
// clients must use exec_command from the ZOA Access session/join API (Terraform → Lambda env).
const defaultExecCommand = "/bin/bash"

// ConnectInteractive waits for the task, starts ECS Exec, and execs session-manager-plugin.
func ConnectInteractive(ctx context.Context, cfg aws.Config, p JoinParams) error {
	session, creds, region, err := prepareConnect(ctx, cfg, p)
	if err != nil {
		return err
	}
	return StartSessionManagerPlugin(region, session, creds)
}
