package reason

import (
	"testing"
)

func TestResolve_WhenFlagSet_ItShouldPreferFlagOverEnv(t *testing.T) {
	t.Setenv(EnvVar, "TEST-99999")
	r, err := Resolve("TEST-12345")
	if err != nil || r != "TEST-12345" {
		t.Fatalf("got %q err=%v", r, err)
	}
}

func TestResolve_WhenPagerDutyInEnv_ItShouldUseEnv(t *testing.T) {
	t.Setenv(EnvVar, "#123456")
	r, err := Resolve("")
	if err != nil || r != "#123456" {
		t.Fatalf("got %q err=%v", r, err)
	}
}

func TestResolve_WhenFlagInvalid_ItShouldNotFallBackToEnv(t *testing.T) {
	t.Setenv(EnvVar, "TEST-99999")
	_, err := Resolve("not-valid")
	if err == nil {
		t.Fatal("expected error for invalid flag")
	}
}
