package actions

import "testing"

func TestGet_WhenTargetTypeFilters_ItShouldHideNonMatchingActions(t *testing.T) {
	t.Cleanup(func() { SetTargetType("") })

	SetTargetType("rc")
	if _, ok := Get("must_gather"); !ok {
		t.Fatal("expected must_gather on rc endpoint")
	}

	SetTargetType("mc")
	if _, ok := Get("must_gather"); !ok {
		t.Fatal("expected must_gather on mc endpoint")
	}
}

func TestList_WhenTargetTypeSet_ItShouldReturnFilteredActions(t *testing.T) {
	t.Cleanup(func() { SetTargetType("") })

	all := len(ListCatalog())
	if all == 0 {
		t.Fatal("expected registered actions")
	}

	SetTargetType("rc")
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
		for _, dt := range meta.TargetTypes {
			if dt == "rc" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %q returned by List() for rc but TargetTypes=%v", meta.Name, meta.TargetTypes)
		}
	}

	SetTargetType("mc")
	mcActions := List()
	if len(mcActions) == 0 {
		t.Fatal("expected actions on mc endpoint")
	}
	for _, a := range mcActions {
		meta := a.Metadata()
		found := false
		for _, dt := range meta.TargetTypes {
			if dt == "mc" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("action %q returned by List() for mc but TargetTypes=%v", meta.Name, meta.TargetTypes)
		}
	}
}
