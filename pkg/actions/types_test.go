package actions

import "testing"

func TestApplyDefaults_WhenParameterMissing_ItShouldFillDefault(t *testing.T) {
	action, ok := GetCatalog("must_gather")
	if !ok {
		t.Fatal("must_gather action not registered")
	}

	params := map[string]string{"gather": "hcp", "cluster_id": "1600392f-9a94-4957-b672-eff8dc2be0bb"}
	ApplyDefaults(action.Metadata(), params)

	if params["skip_must_gather_image"] != "false" {
		t.Errorf("expected skip_must_gather_image=false default, got %q", params["skip_must_gather_image"])
	}
	if params["cluster_id"] != "1600392f-9a94-4957-b672-eff8dc2be0bb" {
		t.Errorf("expected cluster_id unchanged, got %q", params["cluster_id"])
	}
}

func TestApplyDefaults_WhenParameterPresent_ItShouldNotOverride(t *testing.T) {
	action, ok := GetCatalog("must_gather")
	if !ok {
		t.Fatal("must_gather action not registered")
	}

	params := map[string]string{
		"cluster_id":             "1600392f-9a94-4957-b672-eff8dc2be0bb",
		"gather":                 "mc",
		"skip_must_gather_image": "true",
	}
	ApplyDefaults(action.Metadata(), params)

	if params["gather"] != "mc" {
		t.Errorf("expected explicit gather=mc preserved, got %q", params["gather"])
	}
	if params["skip_must_gather_image"] != "true" {
		t.Errorf("expected explicit skip preserved, got %q", params["skip_must_gather_image"])
	}
}
