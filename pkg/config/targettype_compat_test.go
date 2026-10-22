package config

import "testing"

func TestCoalesceTargetTypeEnv_WhenTargetTypeSet_ItShouldPreferTargetType(t *testing.T) {
	t.Setenv("ZOA_TARGET_TYPE", "rc")
	t.Setenv(legacyTargetTypeEnv, "mc")

	if got := coalesceTargetTypeEnv(); got != "rc" {
		t.Fatalf("got %q", got)
	}
}

func TestCoalesceTargetTypeEnv_WhenOnlyLegacyDeploymentTarget_ItShouldUseLegacy(t *testing.T) {
	t.Setenv("ZOA_TARGET_TYPE", "")
	t.Setenv(legacyTargetTypeEnv, "mc")

	if got := coalesceTargetTypeEnv(); got != "mc" {
		t.Fatalf("got %q", got)
	}
}
