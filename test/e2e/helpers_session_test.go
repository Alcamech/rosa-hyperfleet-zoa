//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/onsi/ginkgo/v2"
)

const sessionManagerPluginDir = "/usr/local/sessionmanagerplugin/bin"

var (
	centralProfile = envOrDefault("ZOA_CENTRAL_AWS_PROFILE", "rrp-central")

	sessionDeployOnce sync.Once
	sessionDeployment string
	sessionDeployErr  error
)

func init() {
	// Ginkgo/check helpers run before subprocess zoa calls; match ci/e2e-tests.sh PATH.
	_ = os.Setenv("PATH", e2ePathWithSessionManagerPlugin())
}

// e2ePathWithSessionManagerPlugin matches ci/e2e-tests.sh so boundary exec-script
// smoke finds session-manager-plugin in the rosa-seaplane CI image (and local
// installs that only symlink into /usr/local/sessionmanagerplugin/bin).
func e2ePathWithSessionManagerPlugin() string {
	path := os.Getenv("PATH")
	if path == "" {
		return sessionManagerPluginDir + ":/usr/bin:/usr/local/bin"
	}
	if strings.Contains(path, sessionManagerPluginDir) {
		return path
	}
	return sessionManagerPluginDir + ":" + path
}

// zoaDeploymentName returns ZOA_DEPLOYMENT_NAME (required for every e2e run).
func zoaDeploymentName() string {
	return strings.TrimSpace(os.Getenv("ZOA_DEPLOYMENT_NAME"))
}

const sessionE2ESkipMessage = "Session e2e skipped: ZOA_DEPLOYMENT_NAME must contain eph until int/stage CI has central credentials (temporary gate)"

// sessionE2EEnabled gates session-plane e2e (central profile: deployments, targets, session, exec-script).
//
// TEMPORARY: return true only when ZOA_DEPLOYMENT_NAME contains "eph". Ephemeral CI already
// ships Vault aws_config with rrp-central and Terraform central invoker trust for the build;
// integration/stage standing jobs do not yet assume that central credential path from CI
// (GSM/Vault aws_config + operator profile parity). Names
// like us-east-1 still export ZOA_DEPLOYMENT_NAME for suite wiring but skip session specs until
// we can run the same smoke on nightly-integration/stage without the substring gate.
//
// Remove the strings.Contains(..., "eph") check once standing-env e2e mounts central creds.
func sessionE2EEnabled() bool {
	name := zoaDeploymentName()
	return name != "" && strings.Contains(strings.ToLower(name), "eph")
}

func skipUnlessSessionE2E() {
	if !sessionE2EEnabled() {
		ginkgo.Skip(sessionE2ESkipMessage)
	}
}

// sessionE2EContext holds deployment and target_id for this RC/MC worker.
type sessionE2EContext struct {
	Deployment string
	TargetID   string
}

func resolveSessionE2EContext(targetType string) (sessionE2EContext, error) {
	dep, err := resolveSessionDeployment()
	if err != nil {
		return sessionE2EContext{}, err
	}
	tgtOut, err := runZoaAccess("targets", dep, "-o", "json")
	if err != nil {
		return sessionE2EContext{}, fmt.Errorf("zoa targets: %w\n%s", err, tgtOut)
	}
	rows, err := parseSessionTargetsJSON(tgtOut)
	if err != nil {
		return sessionE2EContext{}, err
	}
	targetID, err := sessionTargetIDForType(rows, targetType)
	if err != nil {
		return sessionE2EContext{}, fmt.Errorf("targets for %s: %w (%v)", targetType, err, rows)
	}
	return sessionE2EContext{Deployment: dep, TargetID: targetID}, nil
}

// sessionTargetIDForType picks the SSM target_id for this worker by target_type
// (rc/mc), regardless of target_id naming (eph-*, regional, mc01, etc.).
func sessionTargetIDForType(rows []sessionTargetRow, targetType string) (string, error) {
	want := strings.ToLower(targetType)
	var matches []string
	for _, row := range rows {
		if strings.ToLower(row.TargetType) == want && row.TargetID != "" {
			matches = append(matches, row.TargetID)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", fmt.Errorf("no target with type %q in list", targetType)
	default:
		return "", fmt.Errorf("ambiguous targets for type %q: %v", targetType, matches)
	}
}

// runZoaAccess runs zoa with ZOA_CENTRAL_AWS_PROFILE (deployments, targets, session subcommands).
func runZoaAccess(args ...string) (string, error) {
	cmd := exec.Command(zoaBin, args...) //nolint:gosec // test helper
	cmd.Env = append(filterEnv(os.Environ(), "AWS_PROFILE", "ZOA_API_URL", "PATH"),
		"AWS_PROFILE="+centralProfile,
		"PATH="+e2ePathWithSessionManagerPlugin(),
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runZoaAccessExpectFailure runs zoa with the central profile and expects a non-zero exit.
func runZoaAccessExpectFailure(args ...string) (string, error) {
	out, err := runZoaAccess(args...)
	if err == nil {
		return out, fmt.Errorf("expected zoa command to fail: %v", args)
	}
	return out, nil
}

func resolveSessionDeployment() (string, error) {
	sessionDeployOnce.Do(func() {
		name := zoaDeploymentName()
		if name == "" {
			sessionDeployErr = fmt.Errorf("ZOA_DEPLOYMENT_NAME is not set")
			return
		}
		sessionDeployment = name
	})
	return sessionDeployment, sessionDeployErr
}

type sessionTargetListJSON struct {
	Items []sessionTargetRow `json:"items"`
	Count int                `json:"count"`
}

type sessionTargetRow struct {
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
	Status     string `json:"status"`
	VpcID      string `json:"vpc_id"`
}

type sessionStartJSON struct {
	SessionID  string `json:"session_id"`
	RawID      string `json:"raw_id"`
	Deployment string `json:"deployment"`
	Target     string `json:"target"`
	Status     string `json:"status"`
	TaskArn    string `json:"task_arn"`
	Region     string `json:"region"`
}

type sessionListJSON struct {
	Items []sessionRow `json:"items"`
	Count int          `json:"count"`
}

// sessionRow mirrors internal/client.Session — the shape of `zoa session list|history -o json`.
// vpc_id / deployment_name may exist in DynamoDB but are not emitted on this CLI contract today.
type sessionRow struct {
	SessionID      string   `json:"session_id"`
	Operator       string   `json:"operator"`
	SignerARN      string   `json:"signer_arn"`
	AccountID      string   `json:"account_id"`
	Reason         string   `json:"reason"`
	TargetCluster  string   `json:"target_cluster"`
	Region         string   `json:"region"`
	TaskArn        string   `json:"task_arn"`
	TaskID         string   `json:"task_id"`
	EcsCluster     string   `json:"ecs_cluster"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"created_at"`
	Deadline       string   `json:"deadline"`
	TerminatedAt   string   `json:"terminated_at"`
	StopReason     string   `json:"stop_reason"`
	ExecSessionIDs []string `json:"exec_session_ids"`
}

func parseSessionTargetsJSON(output string) ([]sessionTargetRow, error) {
	var resp sessionTargetListJSON
	if err := json.Unmarshal([]byte(extractJSON(output)), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func parseSessionStartJSON(output string) (*sessionStartJSON, error) {
	var resp sessionStartJSON
	if err := json.Unmarshal([]byte(extractJSON(output)), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func parseSessionListJSON(output string) ([]sessionRow, error) {
	var resp sessionListJSON
	if err := json.Unmarshal([]byte(extractJSON(output)), &resp); err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func findSessionTargetRow(rows []sessionTargetRow, targetID string) *sessionTargetRow {
	for i := range rows {
		if rows[i].TargetID == targetID {
			return &rows[i]
		}
	}
	return nil
}

func findSessionForTarget(rows []sessionRow, rawID, targetID string) *sessionRow {
	for i := range rows {
		row := &rows[i]
		if row.TargetCluster != targetID {
			continue
		}
		if row.SessionID == rawID || strings.HasSuffix(row.SessionID, "/"+rawID) || strings.Contains(row.SessionID, rawID) {
			return row
		}
	}
	return nil
}

func parseRFC3339(ts string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, ts)
}

func requireNonEmpty(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("session field %q is empty", field)
	}
	return nil
}

func requireARN(field, value, prefix string) error {
	if err := requireNonEmpty(field, value); err != nil {
		return err
	}
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("session field %q: expected %q prefix, got %q", field, prefix, value)
	}
	return nil
}

// validateActiveSessionRecord asserts every field the session API should populate for an active
// boundary session after start (--no-connect). Catches API/CLI regressions that drop
// audit or provisioning metadata from list/history JSON.
func validateActiveSessionRecord(row sessionRow, targetID, rawID, wantReason string) error {
	checks := []func() error{
		func() error {
			if row.SessionID != rawID {
				return fmt.Errorf("session_id: got %q want %q", row.SessionID, rawID)
			}
			return nil
		},
		func() error { return requireNonEmpty("operator", row.Operator) },
		func() error { return requireARN("signer_arn", row.SignerARN, "arn:aws:sts:") },
		func() error { return requireNonEmpty("account_id", row.AccountID) },
		func() error {
			if row.Reason != wantReason {
				return fmt.Errorf("reason: got %q want %q", row.Reason, wantReason)
			}
			return nil
		},
		func() error {
			if row.TargetCluster != targetID {
				return fmt.Errorf("target_cluster: got %q want %q", row.TargetCluster, targetID)
			}
			return nil
		},
		func() error { return requireNonEmpty("region", row.Region) },
		func() error { return requireARN("task_arn", row.TaskArn, "arn:aws:ecs:") },
		func() error { return requireNonEmpty("task_id", row.TaskID) },
		func() error { return requireARN("ecs_cluster", row.EcsCluster, "arn:aws:ecs:") },
		func() error {
			if strings.ToLower(row.Status) != "active" {
				return fmt.Errorf("status: got %q want active", row.Status)
			}
			return nil
		},
		func() error { return requireNonEmpty("created_at", row.CreatedAt) },
		func() error { return requireNonEmpty("deadline", row.Deadline) },
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	created, err := parseRFC3339(row.CreatedAt)
	if err != nil {
		return fmt.Errorf("created_at: %w", err)
	}
	deadline, err := parseRFC3339(row.Deadline)
	if err != nil {
		return fmt.Errorf("deadline: %w", err)
	}
	if !deadline.After(created) {
		return fmt.Errorf("deadline %v must be after created_at %v", deadline, created)
	}
	return nil
}

// validateTerminatedSessionRecord asserts list/history fields after operator terminate.
func validateTerminatedSessionRecord(row sessionRow, targetID, rawID, wantReason string) error {
	checks := []func() error{
		func() error {
			if row.SessionID != rawID {
				return fmt.Errorf("session_id: got %q want %q", row.SessionID, rawID)
			}
			return nil
		},
		func() error { return requireNonEmpty("operator", row.Operator) },
		func() error { return requireARN("signer_arn", row.SignerARN, "arn:aws:sts:") },
		func() error { return requireNonEmpty("account_id", row.AccountID) },
		func() error {
			if row.Reason != wantReason {
				return fmt.Errorf("reason: got %q want %q", row.Reason, wantReason)
			}
			return nil
		},
		func() error {
			if row.TargetCluster != targetID {
				return fmt.Errorf("target_cluster: got %q want %q", row.TargetCluster, targetID)
			}
			return nil
		},
		func() error { return requireNonEmpty("region", row.Region) },
		func() error { return requireARN("task_arn", row.TaskArn, "arn:aws:ecs:") },
		func() error { return requireNonEmpty("task_id", row.TaskID) },
		func() error { return requireARN("ecs_cluster", row.EcsCluster, "arn:aws:ecs:") },
		func() error {
			if strings.ToLower(row.Status) != "terminated" {
				return fmt.Errorf("status: got %q want terminated", row.Status)
			}
			return nil
		},
		func() error { return requireNonEmpty("created_at", row.CreatedAt) },
		func() error { return requireNonEmpty("deadline", row.Deadline) },
		func() error { return requireNonEmpty("terminated_at", row.TerminatedAt) },
		func() error {
			if row.StopReason != "operatorStop" {
				return fmt.Errorf("stop_reason: got %q want operatorStop", row.StopReason)
			}
			return nil
		},
	}
	for _, check := range checks {
		if err := check(); err != nil {
			return err
		}
	}
	return nil
}

func validateSessionStartResponse(started *sessionStartJSON, deployment, targetID, rawID string) error {
	if started.RawID != rawID {
		return fmt.Errorf("raw_id: got %q want %q", started.RawID, rawID)
	}
	if started.Deployment != deployment {
		return fmt.Errorf("deployment: got %q want %q", started.Deployment, deployment)
	}
	if !strings.EqualFold(started.Target, targetID) {
		return fmt.Errorf("target: got %q want %q", started.Target, targetID)
	}
	if err := requireARN("task_arn", started.TaskArn, "arn:aws:ecs:"); err != nil {
		return err
	}
	if err := requireNonEmpty("region", started.Region); err != nil {
		return err
	}
	return nil
}
