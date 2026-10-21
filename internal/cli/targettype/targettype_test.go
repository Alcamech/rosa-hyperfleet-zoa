package targettype

import (
	"testing"
)

func TestResolve_WhenFlagSet_ItShouldPreferFlag(t *testing.T) {
	t.Setenv(EnvVar, "mc")
	got, err := Resolve("rc")
	if err != nil {
		t.Fatal(err)
	}
	if got != RC {
		t.Fatalf("got %q, want rc", got)
	}
}

func TestResolve_WhenOnlyEnvVarSet_ItShouldUseEnvVar(t *testing.T) {
	t.Setenv(EnvVar, "mc")
	got, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != MC {
		t.Fatalf("got %q, want mc", got)
	}
}

func TestResolve_WhenUnset_ItShouldReturnError(t *testing.T) {
	t.Setenv(EnvVar, "")
	t.Setenv("ZOA_DEPLOYMENT_TARGET", "")
	_, err := Resolve("")
	if err == nil {
		t.Fatal("expected error")
	}
}

// ROSAENG-68805: pre-boundary seaplane Terraform env name.
func TestResolve_WhenOnlyLegacyDeploymentTargetSet_ItShouldUseLegacy(t *testing.T) {
	t.Setenv(EnvVar, "")
	t.Setenv("ZOA_DEPLOYMENT_TARGET", "rc")
	got, err := Resolve("")
	if err != nil {
		t.Fatal(err)
	}
	if got != RC {
		t.Fatalf("got %q, want rc", got)
	}
}

func TestResolve_WhenInvalidValue_ItShouldReturnError(t *testing.T) {
	_, err := Resolve("hcp")
	if err == nil {
		t.Fatal("expected error")
	}
}
