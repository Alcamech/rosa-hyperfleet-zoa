//go:build e2e

package e2e

import (
	"strings"
	"testing"
)

// extractBoundaryExecScriptJSON reads the harness line emitted by boundary exec-script e2e scripts
// (see cmd_session_exec_test.go file comment and docs/e2e-testing.md).
func extractBoundaryExecScriptJSON(scriptOut, linePrefix string) string {
	idx := strings.LastIndex(scriptOut, linePrefix)
	if idx < 0 {
		return ""
	}
	rest := scriptOut[idx+len(linePrefix):]
	if nl := strings.IndexByte(rest, '\n'); nl >= 0 {
		rest = rest[:nl]
	}
	return strings.TrimSpace(rest)
}

func TestExtractBoundaryExecScriptJSON_WhenMarkerLinePresentItShouldReturnJSON(t *testing.T) {
	const prefix = "__ZOA_E2E_EXEC_JSON__:"
	out := "MOTD noise\n" + prefix + `{"id":"abc","status":"succeeded"}` + "\n__ZOA_E2E_BOUNDARY_OK__\n"
	got := extractBoundaryExecScriptJSON(out, prefix)
	want := `{"id":"abc","status":"succeeded"}`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractBoundaryExecScriptJSON_WhenMarkerMissingItShouldReturnEmpty(t *testing.T) {
	if got := extractBoundaryExecScriptJSON("no marker", "__ZOA_E2E_EXEC_JSON__:"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
