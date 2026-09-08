package actions

import "testing"

func TestGet_WhenDeploymentTargetFilters_ItShouldHideNonMatchingActions(t *testing.T) {
	t.Cleanup(func() { SetDeploymentTarget("") })

	SetDeploymentTarget("rc")
	if _, ok := Get("must_gather"); !ok {
		t.Fatal("expected must_gather on rc endpoint")
	}

	SetDeploymentTarget("mc")
	if _, ok := Get("must_gather"); !ok {
		t.Fatal("expected must_gather on mc endpoint")
	}
}

func TestList_WhenDeploymentTargetSet_ItShouldReturnFilteredActions(t *testing.T) {
	t.Cleanup(func() { SetDeploymentTarget("") })

	all := len(ListCatalog())
	if all == 0 {
		t.Fatal("expected registered actions")
	}

	SetDeploymentTarget("rc")
	rcActions := List()
	if len(rcActions) == 0 {
		t.Fatal("expected actions on rc endpoint")
	}
	if len(rcActions) > all {
		t.Fatalf("filtered list (%d) must not exceed catalog (%d)", len(rcActions), all)
	}
	for _, a := range rcActions {
		meta := a.Metadata()
		found := false
		for _, dt := range meta.DeploymentTargets {
			if dt == "rc" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %q returned by List() for rc but DeploymentTargets=%v", meta.Name, meta.DeploymentTargets)
		}
	}

	SetDeploymentTarget("mc")
	mcActions := List()
	if len(mcActions) == 0 {
		t.Fatal("expected actions on mc endpoint")
	}
	for _, a := range mcActions {
		meta := a.Metadata()
		found := false
		for _, dt := range meta.DeploymentTargets {
			if dt == "mc" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %q returned by List() for mc but DeploymentTargets=%v", meta.Name, meta.DeploymentTargets)
		}
	}
}
