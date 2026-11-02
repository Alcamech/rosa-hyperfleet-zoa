package cli

import "testing"

func TestParseCompoundSessionIDFromCLIOutput_WhenDispatchedLinePresentItShouldReturnCompoundID(t *testing.T) {
	out := "noise\n✓ us-east-1-eph/1989937d-0889-485d-b64e-facee9e0150d [regional]\nmore"
	got, err := ParseCompoundSessionIDFromCLIOutput(out)
	if err != nil {
		t.Fatalf("ParseCompoundSessionIDFromCLIOutput: %v", err)
	}
	want := "us-east-1-eph/1989937d-0889-485d-b64e-facee9e0150d"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestParseCompoundSessionIDFromCLIOutput_WhenJoinHintPresentItShouldReturnCompoundID(t *testing.T) {
	out := `==> ZOA boundary — Exec ended (ECS task may still be running)
    Re-join session:          zoa session join us-east-1/mc-sess-abc
`
	got, err := ParseCompoundSessionIDFromCLIOutput(out)
	if err != nil {
		t.Fatalf("ParseCompoundSessionIDFromCLIOutput: %v", err)
	}
	if got != "us-east-1/mc-sess-abc" {
		t.Fatalf("got %q", got)
	}
}

func TestParseCompoundSessionIDFromCLIOutput_WhenMissingItShouldError(t *testing.T) {
	_, err := ParseCompoundSessionIDFromCLIOutput("no session markers")
	if err == nil {
		t.Fatal("expected error")
	}
}
