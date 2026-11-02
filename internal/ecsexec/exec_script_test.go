package ecsexec

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseExecScriptExitCode_WhenMarkerPresentItShouldReturnLastCode(t *testing.T) {
	out := "banner noise\n" + ExecScriptExitMarker + "0\nmore\n" + ExecScriptExitMarker + "42\n"
	code, err := parseExecScriptExitCode(out)
	if err != nil {
		t.Fatalf("parseExecScriptExitCode: %v", err)
	}
	if code != 42 {
		t.Fatalf("got exit code %d want 42", code)
	}
}

func TestParseExecScriptExitCode_WhenMarkerMissingItShouldError(t *testing.T) {
	_, err := parseExecScriptExitCode("no marker here")
	if err == nil {
		t.Fatal("expected error when marker missing")
	}
}

func TestParseExecScriptExitCode_WhenMOTDPrecedesMarkerItShouldReturnLastCode(t *testing.T) {
	out := "Hello, operator\n==> ZOA Boundary session\n" + ExecScriptExitMarker + "0\n"
	code, err := parseExecScriptExitCode(out)
	if err != nil {
		t.Fatalf("parseExecScriptExitCode: %v", err)
	}
	if code != 0 {
		t.Fatalf("got exit code %d want 0", code)
	}
}

func TestBoundaryShellReady_WhenBothPromptLinesPresentItShouldReturnTrue(t *testing.T) {
	out := []byte("==> ZOA Boundary\nsessionId:us-east-1/1989937d-0889-485d-b64e-facee9e0150d\nslopezma@zoa:us-east-1/mc01 ~ $ ")
	if !boundaryShellReady(out) {
		t.Fatal("expected ready when sessionId and @zoa prompt lines are present")
	}
}

func TestBoundaryShellReady_WhenMOTDPrecedesPromptsItShouldReturnTrue(t *testing.T) {
	out := []byte("Hello, operator\n\nsessionId:integration/regional-uuid\noperator@zoa:integration/regional ~ $ \n")
	if !boundaryShellReady(out) {
		t.Fatal("expected ready with MOTD before two-line PS1")
	}
}

func TestBoundaryShellReady_WhenOnlySessionLineItShouldReturnFalse(t *testing.T) {
	out := []byte("sessionId:us-east-1/uuid\n")
	if boundaryShellReady(out) {
		t.Fatal("expected not ready without @zoa secondary prompt")
	}
}

func TestBoundaryShellReady_WhenOnlySecondaryPromptItShouldReturnFalse(t *testing.T) {
	out := []byte("sre@zoa:us-east-1/mc01 ~ $ ")
	if boundaryShellReady(out) {
		t.Fatal("expected not ready without sessionId line")
	}
}

func TestExecScriptMaxOutputBytes_WhenZeroItShouldUseDefault(t *testing.T) {
	if got := execScriptMaxOutputBytes(ExecScriptOptions{}); got != DefaultExecScriptMaxOutput {
		t.Fatalf("got %d want default %d", got, DefaultExecScriptMaxOutput)
	}
}

func TestAppendExecScriptTruncationMarker_ItShouldWriteExpectedLine(t *testing.T) {
	var buf bytes.Buffer
	appendExecScriptTruncationMarker(&buf, 1024)
	want := ExecScriptTruncatedMarkerPrefix + "1024\n"
	if buf.String() != want {
		t.Fatalf("got %q want %q", buf.String(), want)
	}
}

func TestExecScriptAppendPTYChunk_WhenUnderLimitItShouldNotTruncate(t *testing.T) {
	var buf bytes.Buffer
	hitLimit := false
	maxOut := 100
	execScriptAppendPTYChunk(&buf, maxOut, &hitLimit, []byte("hello"))
	if hitLimit {
		t.Fatal("expected no truncation under limit")
	}
	if buf.Len() != 5 {
		t.Fatalf("got len %d want 5", buf.Len())
	}
}

func TestExecScriptAppendPTYChunk_WhenChunkExceedsRoomItShouldTruncateWithMarker(t *testing.T) {
	var buf bytes.Buffer
	hitLimit := false
	const maxOut = 10
	execScriptAppendPTYChunk(&buf, maxOut, &hitLimit, []byte("0123456789"))
	if hitLimit {
		t.Fatal("expected exactly at limit without truncation marker yet")
	}
	execScriptAppendPTYChunk(&buf, maxOut, &hitLimit, []byte("extra-bytes"))
	if !hitLimit {
		t.Fatal("expected truncation after exceeding maxOut")
	}
	out := buf.String()
	if !strings.HasPrefix(out, "0123456789") {
		t.Fatalf("expected first 10 bytes preserved, got %q", out[:10])
	}
	if !strings.Contains(out, ExecScriptTruncatedMarkerPrefix+"10") {
		t.Fatalf("expected truncation marker for maxOut=10 in %q", out)
	}
}

func TestExecScriptAppendPTYChunk_WhenAlreadyTruncatedItShouldIgnoreFurtherChunks(t *testing.T) {
	var buf bytes.Buffer
	hitLimit := true
	appendExecScriptTruncationMarker(&buf, 8)
	before := buf.String()
	execScriptAppendPTYChunk(&buf, 8, &hitLimit, []byte("ignored"))
	if buf.String() != before {
		t.Fatal("expected no further capture after hitLimit")
	}
}

// TestExecScriptDefaultSixteenMiBCap_WhenSimulatedWithoutLargeAllocation documents the production
// 16 MiB CLI default without allocating a 16 MiB PTY buffer in tests: constant + marker format,
// synthetic truncated transcript, and overflow via execScriptAppendPTYChunk with a tiny maxOut
// (same function ConnectExecScript uses when maxOut is DefaultExecScriptMaxOutput).
func TestExecScriptDefaultSixteenMiBCap_WhenSimulatedWithoutLargeAllocation(t *testing.T) {
	t.Run("WhenDefaultConstantItShouldBe16777216", func(t *testing.T) {
		if DefaultExecScriptMaxOutput != 16777216 {
			t.Fatalf("DefaultExecScriptMaxOutput = %d, want 16777216 (16 MiB)", DefaultExecScriptMaxOutput)
		}
	})

	t.Run("WhenTruncationMarkerUsesDefaultItShouldShow16777216", func(t *testing.T) {
		got := execScriptTruncatedMarker(DefaultExecScriptMaxOutput)
		want := ExecScriptTruncatedMarkerPrefix + "16777216\n"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})

	t.Run("WhenCaptureExceedsLimitViaSmallMaxItShouldUseSameOverflowLogicAsProduction", func(t *testing.T) {
		var buf bytes.Buffer
		hitLimit := false
		const testMax = 16
		execScriptAppendPTYChunk(&buf, testMax, &hitLimit, bytes.Repeat([]byte("a"), testMax))
		execScriptAppendPTYChunk(&buf, testMax, &hitLimit, []byte("overflow"))
		if !hitLimit {
			t.Fatal("expected truncation")
		}
		if !strings.Contains(buf.String(), ExecScriptTruncatedMarkerPrefix+"16") {
			t.Fatalf("expected marker max=%d in %q", testMax, buf.String())
		}
	})

	t.Run("WhenSyntheticTranscriptHasDefaultMarkerItShouldFailWithTruncatedResult", func(t *testing.T) {
		combined := "sessionId:us-east-1/uuid\nsre@zoa:us-east-1/mc01 ~ $\n" + execScriptTruncatedMarker(DefaultExecScriptMaxOutput)
		result, err, ok := execScriptResultIfTruncated(combined, DefaultExecScriptMaxOutput)
		if !ok {
			t.Fatal("expected truncated=true")
		}
		if err == nil {
			t.Fatal("expected error when output truncated")
		}
		if !result.Truncated {
			t.Fatal("expected result.Truncated")
		}
		if !strings.Contains(err.Error(), "16777216") {
			t.Fatalf("expected default maxOut in error, got %v", err)
		}
	})

	t.Run("WhenCaptureExceededErrorUsesDefaultItShouldMention16777216", func(t *testing.T) {
		err := execScriptCaptureExceededError(DefaultExecScriptMaxOutput)
		if !strings.Contains(err.Error(), "16777216") {
			t.Fatalf("expected 16777216 in error, got %v", err)
		}
	})
}

func TestExecScriptResultIfTruncated_WhenMarkerAbsentItShouldNotTruncate(t *testing.T) {
	_, err, ok := execScriptResultIfTruncated("ok output\n"+ExecScriptExitMarker+"0\n", 1024)
	if ok || err != nil {
		t.Fatalf("expected no truncation, ok=%v err=%v", ok, err)
	}
}
