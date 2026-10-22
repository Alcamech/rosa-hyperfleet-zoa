package api

import "testing"

func TestPathConstants_WhenAudit_ItShouldBePlatformRouteNotUnderTrustedActions(t *testing.T) {
	if PathAudit != V0Prefix+"/audit" {
		t.Errorf("PathAudit = %q, want %q", PathAudit, V0Prefix+"/audit")
	}
	if PathAudit == PathTrustedActions+"/audit" {
		t.Error("PathAudit must not live under PathTrustedActions")
	}
}
