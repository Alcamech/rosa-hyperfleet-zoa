package api

import "strings"

// coalesceReason prefers reason; falls back to legacy JSON field "jira" (pre-reason API clients).
//
// Search: ROSAENG-68805, reason_compat, coalesceReason
//
// TODO(ROSAENG-68805): remove jira fallback after main CLI and boundary images send reason only.
func coalesceReason(reason, jira string) string {
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	return jira
}
