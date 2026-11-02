package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
	"github.com/openshift-online/rosa-seaplane-zoa/internal/ecsexec"
)

func runSessionECSExec(ctx context.Context, deployment, region string, zoaSessionID string, access APIClient, join *client.SessionJoinResponse) error {
	return runSessionECSExecMode(ctx, deployment, region, zoaSessionID, access, join, "", 0)
}

func runSessionECSExecScript(ctx context.Context, deployment, region string, zoaSessionID string, access APIClient, join *client.SessionJoinResponse, script string, timeout time.Duration, maxOutputBytes int) error {
	if err := ecsexec.RequireSessionManagerPlugin(); err != nil {
		return err
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	if maxOutputBytes <= 0 {
		maxOutputBytes = ecsexec.DefaultExecScriptMaxOutput
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cfg, err := awsConfigFromJoin(execCtx, region, join)
	if err != nil {
		return err
	}
	opts := ecsexec.ExecScriptOptions{MaxOutputBytes: maxOutputBytes}
	result, err := ecsexec.ConnectExecScript(execCtx, cfg, execScriptJoinParams(ctx, region, zoaSessionID, access, join), script, opts)
	if writeErr := ecsexec.WriteExecScriptOutput(os.Stdout, result); writeErr != nil {
		return writeErr
	}
	if result.Truncated {
		fmt.Fprintf(os.Stderr, "Warning: exec-script output exceeded %d bytes; CLI transcript truncated (see %s in stdout). Full transcript is in CloudWatch Exec logs. Split automation into smaller --exec-script runs or raise --exec-script-max-output.\n",
			maxOutputBytes, ecsexec.ExecScriptTruncatedMarkerPrefix)
	}
	return err
}

func execScriptJoinParams(ctx context.Context, region, zoaSessionID string, access APIClient, join *client.SessionJoinResponse) ecsexec.JoinParams {
	return ecsexec.JoinParams{
		Region:        region,
		Cluster:       join.ClusterArn,
		TaskARN:       join.TaskArn,
		ContainerName: join.ContainerName,
		Command:       join.ExecCommand,
		OnExecSession: func(execSessionID string) error {
			if access == nil {
				fmt.Fprintln(os.Stderr, "Warning: exec session not registered (access client not configured); idle reaper still uses AWS exec logs")
				return nil
			}
			if err := access.SessionExecAttached(ctx, zoaSessionID, execSessionID); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not register exec session with Access API (%v); continuing exec-script session\n", err)
			}
			return nil
		},
	}
}

func runSessionECSExecMode(ctx context.Context, deployment, region string, zoaSessionID string, access APIClient, join *client.SessionJoinResponse, execScript string, execScriptTimeout time.Duration) error {
	if join.TaskArn == "" {
		return fmt.Errorf("session has no task ARN")
	}
	if join.ExecCommand == "" {
		return fmt.Errorf("join response missing exec_command")
	}
	if join.ExecCredentials == nil || join.ExecCredentials.AccessKeyID == "" {
		return fmt.Errorf("join response missing exec_credentials")
	}

	if execScript != "" {
		return runSessionECSExecScript(ctx, deployment, region, zoaSessionID, access, join, execScript, execScriptTimeout, ecsexec.DefaultExecScriptMaxOutput)
	}

	cfg, err := awsConfigFromJoin(ctx, region, join)
	if err != nil {
		return err
	}
	return ecsexec.ConnectInteractive(ctx, cfg, execScriptJoinParams(ctx, region, zoaSessionID, access, join))
}

func awsConfigFromJoin(ctx context.Context, region string, join *client.SessionJoinResponse) (aws.Config, error) {
	reg := region
	if join.Region != "" {
		reg = join.Region
	}
	if reg == "" {
		return aws.Config{}, fmt.Errorf("session join response missing region")
	}
	c := join.ExecCredentials
	return awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(reg),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.AccessKeyID,
			c.SecretAccessKey,
			c.SessionToken,
		)),
	)
}

func sessionJoinRegion(opts *GlobalOptions, join *client.SessionJoinResponse) string {
	if join.Region != "" {
		return join.Region
	}
	if opts != nil && opts.Region != "" {
		return opts.Region
	}
	return ""
}
