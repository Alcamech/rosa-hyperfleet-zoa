package sessionplane

import (
	"os"
	"strings"
)

// SessionE2ESkipMessage is the message used when session-plane CLI e2e and
// dependent metrics specs are skipped (standing int/stage until central creds parity).
const SessionE2ESkipMessage = "Session e2e skipped: ZOA_DEPLOYMENT_NAME must contain eph until int/stage CI has central credentials (temporary gate)"

// ZoaDeploymentName returns ZOA_DEPLOYMENT_NAME (required for every e2e run).
func ZoaDeploymentName() string {
	return strings.TrimSpace(os.Getenv("ZOA_DEPLOYMENT_NAME"))
}

// SessionE2EEnabled gates session-plane e2e (central profile: deployments, targets, session, exec-script)
// and monitoring specs that require the same Access/boundary traffic.
//
// TEMPORARY: return true only when ZOA_DEPLOYMENT_NAME contains "eph". Ephemeral CI already
// ships Vault aws_config with rrp-central and Terraform central invoker trust for the build;
// integration/stage standing jobs do not yet assume that central credential path from CI
// (GSM/Vault aws_config + operator profile parity). Names
// like us-east-1 still export ZOA_DEPLOYMENT_NAME for suite wiring but skip session specs until
// we can run the same smoke on nightly-integration/stage without the substring gate.
//
// Remove the strings.Contains(..., "eph") check once standing-env e2e mounts central creds.
func SessionE2EEnabled() bool {
	name := ZoaDeploymentName()
	return name != "" && strings.Contains(strings.ToLower(name), "eph")
}
