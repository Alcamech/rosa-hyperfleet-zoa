package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintSessionExecEndedHint_WhenCompoundIDSetItShouldIncludeTerminateAndJoinLines(t *testing.T) {
	var buf bytes.Buffer
	printSessionExecEndedHint(&buf, "us-east-1-eph", "us-east-1-eph/sess-abc", "")

	out := buf.String()
	if !strings.Contains(out, "==> ZOA boundary — Exec ended (ECS task may still be running)") {
		t.Fatalf("missing exec ended header:\n%s", out)
	}
	if !strings.Contains(out, "zoa session terminate us-east-1-eph/sess-abc") {
		t.Fatalf("missing terminate line:\n%s", out)
	}
	if !strings.Contains(out, "zoa session join us-east-1-eph/sess-abc") {
		t.Fatalf("missing join line:\n%s", out)
	}
	if !strings.Contains(out, "zoa session list us-east-1-eph") {
		t.Fatalf("missing list line:\n%s", out)
	}
	if !strings.Contains(out, "zoa session history us-east-1-eph") {
		t.Fatalf("missing history line:\n%s", out)
	}
}

func TestPrintSessionExecEndedHint_WhenDeadlineRFC3339ItShouldFormatHardTerminationUTC(t *testing.T) {
	var buf bytes.Buffer
	printSessionExecEndedHint(&buf, "dep", "dep/id", "2026-10-30T14:02:00Z")

	out := buf.String()
	want := "Hard termination (UTC): 2026-10-30 14:02 UTC"
	if !strings.Contains(out, want) {
		t.Fatalf("expected %q in:\n%s", want, out)
	}
}

func TestPrintSessionExecEndedHint_WhenCompoundIDEmptyItShouldPrintNothing(t *testing.T) {
	var buf bytes.Buffer
	printSessionExecEndedHint(&buf, "dep", "", "2026-10-30T14:02:00Z")
	if buf.Len() != 0 {
		t.Fatalf("expected empty output, got:\n%s", buf.String())
	}
}
