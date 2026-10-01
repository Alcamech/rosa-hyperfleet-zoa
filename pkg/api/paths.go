package api

// API version prefix for all ZOA HTTP resources (TA plane and Access plane).
const V0Prefix = "/api/v0"

// Trusted Action routes (HANDLER_MODE=api).
const (
	PathTrustedActions     = V0Prefix + "/trusted-actions"
	PathTrustedActionsRuns = V0Prefix + "/trusted-actions/runs"
	// PathLegacyAudit is the pre-v0.5 audit URL; main-branch zoa CLI still calls it (ROSAENG-60291).
	PathLegacyAudit = PathTrustedActions + "/audit"
)

// Platform routes (HANDLER_MODE=api) — cross-cutting; not under /trusted-actions.
const (
	PathAudit = V0Prefix + "/audit"
)

// Access / boundary routes (HANDLER_MODE=access). Same v0 prefix and error envelope as TA routes.
const (
	PathTargets            = V0Prefix + "/targets"
	PathSessions           = V0Prefix + "/sessions"
	PathSessionsStart      = V0Prefix + "/sessions/start"
	PathSessionsTerminate  = V0Prefix + "/sessions/terminate/"
	PathSessionsJoin       = V0Prefix + "/sessions/join/"
	PathSessionsExecAttach = V0Prefix + "/sessions/exec-attached/"
	PathApprove            = V0Prefix + "/approve/"
	PathReject             = V0Prefix + "/reject/"
)

// Operational routes (both handler modes).
const (
	PathHealth  = "/health"
	PathVersion = "/version"
)
