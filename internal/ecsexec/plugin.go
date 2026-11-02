package ecsexec

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const sessionManagerPlugin = "session-manager-plugin"

var execSessionManagerPlugin = syscall.Exec

// NewSessionManagerPluginCommand builds a session-manager-plugin command for subprocess use (exec-script ECS Exec).
func NewSessionManagerPluginCommand(region string, session *Session, creds aws.Credentials) (*exec.Cmd, error) {
	pluginPath, args, err := sessionManagerPluginArgs(region, session)
	if err != nil {
		return nil, err
	}
	env, err := pluginEnv(region, creds)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(pluginPath, args[1:]...) //nolint:gosec // argv built from ECS ExecuteCommand response
	cmd.Env = env
	return cmd, nil
}

// StartSessionManagerPlugin hands the terminal to session-manager-plugin (process replace).
func StartSessionManagerPlugin(region string, session *Session, creds aws.Credentials) error {
	pluginPath, args, err := sessionManagerPluginArgs(region, session)
	if err != nil {
		return err
	}
	env, err := pluginEnv(region, creds)
	if err != nil {
		return err
	}
	return execSessionManagerPlugin(pluginPath, args, env)
}

func sessionManagerPluginArgs(region string, session *Session) (pluginPath string, args []string, err error) {
	pluginPath, err = exec.LookPath(sessionManagerPlugin)
	if err != nil {
		return "", nil, fmt.Errorf(
			"session-manager-plugin not found in PATH.\n"+
				"Install: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html\n"+
				"Original error: %w", err)
	}

	ssmEndpoint := fmt.Sprintf("https://ssm.%s.amazonaws.com", region)
	paramsJSON := "{}"
	if session.Target != "" {
		params, _ := json.Marshal(map[string]string{"Target": session.Target})
		paramsJSON = string(params)
	}

	args = []string{
		pluginPath,
		string(session.RawSession),
		region,
		"StartSession",
		"",
		paramsJSON,
		ssmEndpoint,
	}
	return pluginPath, args, nil
}

func pluginEnv(region string, creds aws.Credentials) ([]string, error) {
	if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
		return nil, fmt.Errorf("AWS credentials required for ECS Exec (use your Jump/Central AWS login; the CLI assumes into the deployment account)")
	}

	stripKeys := map[string]bool{
		"AWS_ACCESS_KEY_ID": true, "AWS_SECRET_ACCESS_KEY": true, "AWS_SESSION_TOKEN": true,
		"AWS_SECURITY_TOKEN": true, "AWS_PROFILE": true, "AWS_DEFAULT_PROFILE": true,
		"AWS_SHARED_CREDENTIALS_FILE": true, "AWS_CONFIG_FILE": true,
		"AWS_REGION": true, "AWS_DEFAULT_REGION": true,
	}

	env := make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if !stripKeys[key] {
			env = append(env, entry)
		}
	}
	env = append(env,
		"AWS_ACCESS_KEY_ID="+creds.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY="+creds.SecretAccessKey,
	)
	if creds.SessionToken != "" {
		env = append(env, "AWS_SESSION_TOKEN="+creds.SessionToken)
	}
	if region != "" {
		env = append(env, "AWS_REGION="+region, "AWS_DEFAULT_REGION="+region)
	}
	return env, nil
}
