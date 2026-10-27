//go:build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var (
	centralProfile = envOrDefault("ZOA_CENTRAL_AWS_PROFILE", "rrp-central")

	accessDeployOnce sync.Once
	accessDeployment string
	accessDeployErr  error
)

// zoaDeploymentName returns ZOA_DEPLOYMENT_NAME (required for every e2e run).
func zoaDeploymentName() string {
	return strings.TrimSpace(os.Getenv("ZOA_DEPLOYMENT_NAME"))
}

// accessE2EEnabled gates Access-plane smoke (central AWS profile: deployments, targets, session).
//
// TEMPORARY: return true only when ZOA_DEPLOYMENT_NAME contains "eph". Ephemeral CI already
// ships Vault aws_config with rrp-central and Terraform central invoker trust for the build;
// integration/stage standing jobs do not yet assume that central credential path from CI
// (GSM/Vault aws_config + operator profile parity). Names
// like us-east-1 still export ZOA_DEPLOYMENT_NAME for suite wiring but skip Access specs until
// we can run the same smoke on nightly-integration/stage without the substring gate.
//
// Remove the strings.Contains(..., "eph") check once standing-env e2e mounts central creds.
func accessE2EEnabled() bool {
	name := zoaDeploymentName()
	return name != "" && strings.Contains(strings.ToLower(name), "eph")
}

// accessTargetIDForType picks the SSM target_id for this worker by target_type
// (rc/mc), regardless of target_id naming (eph-*, regional, mc01, etc.).
func accessTargetIDForType(rows []accessTargetRow, targetType string) (string, error) {
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

func runZoaAccess(args ...string) (string, error) {
	cmd := exec.Command(zoaBin, args...) //nolint:gosec // test helper
	cmd.Env = append(filterEnv(os.Environ(), "AWS_PROFILE", "ZOA_API_URL"),
		"AWS_PROFILE="+centralProfile,
	)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func resolveAccessDeployment() (string, error) {
	accessDeployOnce.Do(func() {
		name := zoaDeploymentName()
		if name == "" {
			accessDeployErr = fmt.Errorf("ZOA_DEPLOYMENT_NAME is not set")
			return
		}
		accessDeployment = name
	})
	return accessDeployment, accessDeployErr
}

type accessTargetListJSON struct {
	Items []accessTargetRow `json:"items"`
	Count int               `json:"count"`
}

type accessTargetRow struct {
	TargetID   string `json:"target_id"`
	TargetType string `json:"target_type"`
	Status     string `json:"status"`
}

type sessionStartJSON struct {
	SessionID string `json:"session_id"`
	RawID     string `json:"raw_id"`
	Target    string `json:"target"`
	Status    string `json:"status"`
}

type sessionListJSON struct {
	Items []sessionRow `json:"items"`
	Count int          `json:"count"`
}

type sessionRow struct {
	SessionID     string `json:"session_id"`
	TargetCluster string `json:"target_cluster"`
	Status        string `json:"status"`
}

func parseAccessTargetsJSON(output string) ([]accessTargetRow, error) {
	var resp accessTargetListJSON
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

func findSessionForTarget(rows []sessionRow, rawID, accessTarget string) *sessionRow {
	for i := range rows {
		row := &rows[i]
		if row.TargetCluster != accessTarget {
			continue
		}
		if row.SessionID == rawID || strings.HasSuffix(row.SessionID, "/"+rawID) || strings.Contains(row.SessionID, rawID) {
			return row
		}
	}
	return nil
}
