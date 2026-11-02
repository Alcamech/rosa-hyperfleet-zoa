package ecsexec

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/creack/pty"
)

const (
	// ExecScriptExitMarker is appended by ConnectExecScript after the caller's script runs.
	ExecScriptExitMarker = "__ZOA_EXEC_SCRIPT_EXIT:"
	// ExecScriptTruncatedMarkerPrefix begins a line written when laptop-side capture hits MaxOutputBytes.
	ExecScriptTruncatedMarkerPrefix = "__ZOA_EXEC_SCRIPT_TRUNCATED:max="
	// DefaultExecScriptMaxOutput is the default laptop-side PTY capture limit (full transcript stays in CloudWatch).
	DefaultExecScriptMaxOutput = 16 << 20 // 16 MiB
)

// ExecScriptOptions configures ConnectExecScript. Zero value uses defaults.
type ExecScriptOptions struct {
	// MaxOutputBytes caps combined PTY bytes retained in memory and returned in ExecScriptResult.Output.
	// Zero or negative uses DefaultExecScriptMaxOutput.
	MaxOutputBytes int
}

func execScriptMaxOutputBytes(opts ExecScriptOptions) int {
	if opts.MaxOutputBytes > 0 {
		return opts.MaxOutputBytes
	}
	return DefaultExecScriptMaxOutput
}

// ExecScriptResult holds combined remote shell output from an exec-script ECS Exec session.
type ExecScriptResult struct {
	Output    string
	ExitCode  int
	Truncated bool
}

var execScriptExitLine = regexp.MustCompile(`(?m)` + regexp.QuoteMeta(ExecScriptExitMarker) + `(\d+)\s*`)

// boundaryShellPromptPrimary matches the first PS1 line (sessionId:deployment/uuid).
// boundaryShellPromptSecondary matches the second line (operator@zoa:deployment/target … ~ $).
// ConnectExecScript waits for both before writing the script so MOTD/banner bytes are not
// mistaken for typed input (race that corrupts the first command and yields exit 1).
var (
	boundaryShellPromptPrimary   = regexp.MustCompile(`sessionId:[^\r\n]+`)
	boundaryShellPromptSecondary = regexp.MustCompile(`@zoa:[^\r\n]+ ~ \$`)
)

const defaultExecScriptPromptWait = 2 * time.Minute

func boundaryShellReady(output []byte) bool {
	s := output
	return boundaryShellPromptPrimary.Match(s) && boundaryShellPromptSecondary.Match(s)
}

func execScriptTruncatedMarker(maxBytes int) string {
	return ExecScriptTruncatedMarkerPrefix + strconv.Itoa(maxBytes) + "\n"
}

func appendExecScriptTruncationMarker(buf *bytes.Buffer, maxBytes int) {
	_, _ = buf.WriteString(execScriptTruncatedMarker(maxBytes))
}

// execScriptAppendPTYChunk appends PTY bytes to outBuf up to maxOut. When capture exceeds maxOut,
// writes ExecScriptTruncatedMarkerPrefix once and sets *hitLimit; later chunks are not retained.
func execScriptAppendPTYChunk(outBuf *bytes.Buffer, maxOut int, hitLimit *bool, chunk []byte) {
	if *hitLimit {
		return
	}
	room := maxOut - outBuf.Len()
	if room <= 0 {
		*hitLimit = true
		appendExecScriptTruncationMarker(outBuf, maxOut)
		return
	}
	if len(chunk) > room {
		_, _ = outBuf.Write(chunk[:room])
		*hitLimit = true
		appendExecScriptTruncationMarker(outBuf, maxOut)
		return
	}
	_, _ = outBuf.Write(chunk)
}

func execScriptCaptureExceededError(maxOut int) error {
	return fmt.Errorf(
		"exec script captured output exceeded %d bytes (truncated in CLI output; full transcript in CloudWatch)",
		maxOut,
	)
}

func execScriptResultIfTruncated(combined string, maxOut int) (ExecScriptResult, error, bool) {
	if !strings.Contains(combined, ExecScriptTruncatedMarkerPrefix) {
		return ExecScriptResult{}, nil, false
	}
	return ExecScriptResult{
		Output:    combined,
		ExitCode:  -1,
		Truncated: true,
	}, execScriptCaptureExceededError(maxOut), true
}

// ConnectExecScript opens ECS Exec via session-manager-plugin as a subprocess with a PTY,
// runs script inside the boundary shell, and returns combined output plus the script exit code.
//
// Exec-script mode uses the same ECS Exec audit path as ConnectInteractive (new SSM exec session id per
// connect; CloudWatch stream ecs-execute-command-<id>). ExecuteCommand uses the interactive
// login shell from Access; the automation script is typed into that PTY after the shell prompt
// appears so CloudWatch captures MOTD, prompts, echoed commands, and command output.
//
// The wrapper sets TERM=xterm-256color because CI/automation often has no TTY; the remote
// login shell still expects a capable terminal. Exit status is read from the last
// ExecScriptExitMarker line after the subshell runs.
//
// Output beyond MaxOutputBytes is truncated in ExecScriptResult.Output with ExecScriptTruncatedMarkerPrefix;
// the CLI should treat Truncated as failure. CloudWatch retains the full Exec transcript.
func ConnectExecScript(ctx context.Context, cfg aws.Config, p JoinParams, script string, opts ExecScriptOptions) (ExecScriptResult, error) {
	if strings.TrimSpace(script) == "" {
		return ExecScriptResult{}, fmt.Errorf("exec script is empty")
	}

	maxOut := execScriptMaxOutputBytes(opts)

	session, creds, region, err := prepareConnect(ctx, cfg, p)
	if err != nil {
		return ExecScriptResult{}, err
	}

	cmd, err := NewSessionManagerPluginCommand(region, session, creds)
	if err != nil {
		return ExecScriptResult{}, err
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return ExecScriptResult{}, fmt.Errorf("start session-manager-plugin with pty: %w", err)
	}
	defer func() { _ = ptmx.Close() }()

	var outBuf bytes.Buffer
	promptSeen := make(chan struct{}, 1)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		buf := make([]byte, 4096)
		notified := false
		hitLimit := false
		for {
			n, readErr := ptmx.Read(buf)
			if n > 0 {
				execScriptAppendPTYChunk(&outBuf, maxOut, &hitLimit, buf[:n])
				if !notified && boundaryShellReady(outBuf.Bytes()) {
					notified = true
					promptSeen <- struct{}{}
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	promptWait := defaultExecScriptPromptWait
	if deadline, ok := ctx.Deadline(); ok {
		if remaining := time.Until(deadline); remaining > 0 && remaining < promptWait {
			promptWait = remaining
		}
	}
	select {
	case <-promptSeen:
	case <-time.After(promptWait):
		<-readDone
		return ExecScriptResult{Output: outBuf.String()}, fmt.Errorf(
			"timeout waiting for boundary shell prompt (sessionId:… and @zoa:… ~ $); captured %d bytes",
			outBuf.Len(),
		)
	case <-ctx.Done():
		<-readDone
		return ExecScriptResult{Output: outBuf.String()}, ctx.Err()
	}

	wrapped := fmt.Sprintf(
		"export TERM=xterm-256color\n(%s)\necho %s$?\nexit\n",
		script, ExecScriptExitMarker,
	)
	if _, err := io.WriteString(ptmx, wrapped); err != nil {
		<-readDone
		return ExecScriptResult{Output: outBuf.String()}, fmt.Errorf("write exec script to pty: %w", err)
	}

	waitErr := waitCommand(ctx, cmd)
	<-readDone

	combined := outBuf.String()

	if result, err, truncated := execScriptResultIfTruncated(combined, maxOut); truncated {
		return result, err
	}

	exitCode, parseErr := parseExecScriptExitCode(combined)
	if parseErr != nil {
		if waitErr != nil {
			return ExecScriptResult{Output: combined}, fmt.Errorf("exec script: %w (output parse: %v)", waitErr, parseErr)
		}
		return ExecScriptResult{Output: combined}, parseErr
	}
	if waitErr != nil && !isExitError(waitErr) {
		return ExecScriptResult{Output: combined, ExitCode: exitCode}, waitErr
	}
	if exitCode != 0 {
		return ExecScriptResult{Output: combined, ExitCode: exitCode}, fmt.Errorf("exec script exit status %d", exitCode)
	}
	return ExecScriptResult{Output: combined, ExitCode: exitCode}, nil
}

func prepareConnect(ctx context.Context, cfg aws.Config, p JoinParams) (*Session, aws.Credentials, string, error) {
	if p.ContainerName == "" {
		p.ContainerName = "zoa-boundary"
	}
	if p.Command == "" {
		p.Command = defaultExecCommand
	}

	clusterName, taskID, err := ClusterAndTaskFromTaskARN(p.TaskARN)
	if err != nil {
		return nil, aws.Credentials{}, "", err
	}
	if p.Cluster != "" {
		clusterName = ClusterNameFromARN(p.Cluster)
	}

	ecsClient := NewClient(cfg, clusterName)

	if !p.NoWait {
		if err := runWithSpinner(ctx, "boundary task starting", func(ctx context.Context) error {
			return ecsClient.WaitForRunning(ctx, taskID)
		}); err != nil {
			return nil, aws.Credentials{}, "", fmt.Errorf("task not running: %w", err)
		}
		if err := runWithSpinner(ctx, "ECS Exec agent", func(ctx context.Context) error {
			return ecsClient.WaitForExecAgent(ctx, taskID, p.ContainerName, 30*time.Second)
		}); err != nil {
			return nil, aws.Credentials{}, "", fmt.Errorf("exec agent not ready: %w", err)
		}
	}

	var session *Session
	if err := runWithSpinner(ctx, "opening ECS Exec", func(ctx context.Context) error {
		var execErr error
		session, execErr = ecsClient.ExecuteCommand(ctx, taskID, p.ContainerName, p.Command)
		return execErr
	}); err != nil {
		return nil, aws.Credentials{}, "", err
	}

	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		return nil, aws.Credentials{}, "", fmt.Errorf("retrieve AWS credentials for exec: %w", err)
	}

	if p.OnExecSession != nil && session != nil {
		_ = p.OnExecSession(session.SessionID)
	}

	region := p.Region
	if region == "" {
		region = cfg.Region
	}
	return session, creds, region, nil
}

func waitCommand(ctx context.Context, cmd *exec.Cmd) error {
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		_ = cmd.Process.Kill()
		<-done
		return ctx.Err()
	}
}

func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func parseExecScriptExitCode(output string) (int, error) {
	matches := execScriptExitLine.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return -1, fmt.Errorf("exec script marker %q not found in exec output (len=%d)", ExecScriptExitMarker, len(output))
	}
	last := matches[len(matches)-1]
	if len(last) < 2 {
		return -1, fmt.Errorf("invalid exec script marker match")
	}
	code, err := strconv.Atoi(last[1])
	if err != nil {
		return -1, fmt.Errorf("parse exec script exit code: %w", err)
	}
	return code, nil
}

// RequireSessionManagerPlugin returns an error when the plugin binary is not on PATH.
func RequireSessionManagerPlugin() error {
	_, err := exec.LookPath(sessionManagerPlugin)
	if err != nil {
		return fmt.Errorf(
			"session-manager-plugin not found in PATH (required for ECS Exec with --exec-script): %w",
			err,
		)
	}
	return nil
}

// WriteExecScriptOutput prints exec-script output to stdout when not in JSON mode.
func WriteExecScriptOutput(w io.Writer, result ExecScriptResult) error {
	if w == nil {
		w = os.Stdout
	}
	_, err := io.Copy(w, strings.NewReader(result.Output))
	return err
}
