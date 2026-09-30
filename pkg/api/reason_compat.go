package api

import "strings"

// coalesceReason prefers reason; falls back to legacy JSON field "jira" (pre-reason API clients).
// TODO(ROSAENG-60291): remove jira fallback after main CLI and boundary images send reason only.
func coalesceReason(reason, jira string) string {
	if strings.TrimSpace(reason) != "" {
		return reason
	}
	return jira
}
