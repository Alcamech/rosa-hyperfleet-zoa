package api

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/store"
)

// BridgeOutcome describes how ResolveIdentity attributed a boundary task-shaped caller.
type BridgeOutcome string

const (
	BridgeOutcomeSkipped BridgeOutcome = "skipped" // non-boundary caller or non-task session name; no Dynamo lookup
	BridgeOutcomeHit     BridgeOutcome = "hit"
	BridgeOutcomeMiss    BridgeOutcome = "miss"
	BridgeOutcomeError   BridgeOutcome = "error" // Dynamo failure; TA still runs; operator = ECS task id
)

// ExtractSREIdentity parses an STS assumed-role ARN to extract the SRE username
// and role name.
//
// ARN format: arn:aws:sts::ACCOUNT:assumed-role/ROLE_NAME/SESSION_NAME
// Example: "arn:aws:sts::123456:assumed-role/sre-role/slopezma" → ("slopezma", "sre-role")
func ExtractSREIdentity(operatorARN string) (username, role string, err error) {
	if operatorARN == "" {
		return "", "", fmt.Errorf("empty operator ARN")
	}

	parts := strings.Split(operatorARN, "/")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid assumed-role ARN format: %q (expected arn:.../role/session)", operatorARN)
	}

	if !strings.Contains(operatorARN, "assumed-role") {
		return "", "", fmt.Errorf("not an assumed-role ARN: %q", operatorARN)
	}

	username = parts[len(parts)-1]
	role = parts[len(parts)-2]
	if username == "" {
		return "", "", fmt.Errorf("empty session name in ARN: %q", operatorARN)
	}
	return username, role, nil
}

// BoundaryTaskRoleName is the IAM role name on boundary ECS tasks for this API target.
// Same string Terraform uses for the ECS task role ({cluster_id}-zoa-boundary-task).
// Used for attribution (bridge eligibility), not for authorization — see docs/boundary/identity-bridge.md.
func BoundaryTaskRoleName(targetCluster string) string {
	if targetCluster == "" {
		return ""
	}
	return targetCluster + "-zoa-boundary-task"
}

// IsBoundaryTaskCaller reports whether signerARN is the configured boundary task role for this target.
func IsBoundaryTaskCaller(signerARN, boundaryTaskRoleName string) bool {
	if boundaryTaskRoleName == "" {
		return false
	}
	_, role, err := ExtractSREIdentity(signerARN)
	if err != nil {
		return false
	}
	return role == boundaryTaskRoleName
}

type IdentityResult struct {
	Operator  string        // Session owner id from bridge, or ECS task id when bridge miss/error
	SessionID string        // Boundary session ID (empty when bridge did not run or did not hit)
	Outcome   BridgeOutcome // Bridge lookup result for task-shaped callers
}

// ResolveIdentity sets operator and session_id for API audit and executions (attribution only).
//
// Design: docs/boundary/identity-bridge.md
//
// Identity input is always the SigV4-authenticated IAM userArn from the Lambda authorizer, never a client header.
//
// Authorization (invoke access) is Function URL IAM policy in Terraform, not this function.
// Attribution paths:
//   - Boundary task role for this target + ECS task id session name → GetByTaskID (hit / miss / error).
//   - Other authenticated caller shapes → operator = STS session name; session_id empty.
func ResolveIdentity(ctx context.Context, signerARN, boundaryTaskRoleName string, sessionStore store.SessionStore) (*IdentityResult, error) {
	if signerARN == "" {
		return nil, fmt.Errorf("empty caller ARN")
	}

	sessionName, roleName, err := ExtractSREIdentity(signerARN)
	if err != nil {
		return nil, err
	}

	boundaryCaller := boundaryTaskRoleName != "" && roleName == boundaryTaskRoleName
	if sessionStore != nil && boundaryCaller && looksLikeECSTaskID(sessionName) {
		session, err := sessionStore.GetByTaskID(ctx, sessionName)
		if err != nil {
			return &IdentityResult{
				Operator: sessionName,
				Outcome:  BridgeOutcomeError,
			}, nil
		}
		if session != nil {
			return &IdentityResult{
				Operator:  session.Operator,
				SessionID: session.SessionID,
				Outcome:   BridgeOutcomeHit,
			}, nil
		}
		return &IdentityResult{
			Operator: sessionName,
			Outcome:  BridgeOutcomeMiss,
		}, nil
	}

	return &IdentityResult{
		Operator: sessionName,
		Outcome:  BridgeOutcomeSkipped,
	}, nil
}

// looksLikeECSTaskID reports whether the STS session name has the shape ECS uses for task ids (32-char hex).
// Used together with boundary task role name matching; hex alone does not imply a boundary caller.
func looksLikeECSTaskID(s string) bool {
	if len(s) != 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}
