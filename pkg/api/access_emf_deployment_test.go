package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/store"
)

func TestAccessHandler_WhenTargetList_ItShouldEmitHTTPMetricsWithDeploymentName(t *testing.T) {
	targets := &mockTargetStoreAccess{
		targets: []*store.Target{{TargetID: "eph-test-mc01", TargetType: "MC"}},
	}
	h := testAccessHandler(&mockSessionStoreAccess{}, targets)

	line := captureStdoutAccessEMF(func() {
		rr := doAccessRequest(h, "GET", "/api/v0/targets", nil, accessHeaders())
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
	if !strings.Contains(line, "us-east-1-eph-test") {
		t.Fatalf("expected deployment name in EMF: %s", line)
	}
}
