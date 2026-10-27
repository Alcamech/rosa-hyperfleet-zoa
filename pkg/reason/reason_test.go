package reason

import "testing"

func TestNormalize_WhenJiraKey_ItShouldAccept(t *testing.T) {
	v, err := Normalize("TEST-12345")
	if err != nil || v != "TEST-12345" {
		t.Fatalf("got %q err=%v", v, err)
	}
}

func TestNormalize_WhenPagerDutyIncident_ItShouldAccept(t *testing.T) {
	v, err := Normalize("#123456")
	if err != nil || v != "#123456" {
		t.Fatalf("got %q err=%v", v, err)
	}
}

func TestNormalize_WhenInvalid_ItShouldError(t *testing.T) {
	_, err := Normalize("3220876")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = Normalize("Jira TEST-12345")
	if err == nil {
		t.Fatal("expected error for Jira prefix")
	}
}

func TestRequire_WhenEmpty_ItShouldError(t *testing.T) {
	_, err := Require("")
	if err == nil {
		t.Fatal("expected error")
	}
}
