package api

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/store"
)

func captureStdoutAccessEMF(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestAccessHandler_WhenSessionStart_ItShouldEmitHTTPMetricsWithTargetCluster(t *testing.T) {
	targets := &mockTargetStoreAccess{
		targets: []*store.Target{{
			TargetID:          "eph-test-mc01",
			DeploymentName:    "us-east-1",
			VpcId:             "vpc-123",
			SubnetIds:         "subnet-a",
			SecurityGroupId:   "sg-1",
			EcsClusterArn:     "arn:aws:ecs:us-east-1:123456789012:cluster/test",
			TaskDefinitionArn: "arn:aws:ecs:us-east-1:123456789012:task-definition/td:1",
			Region:            "us-east-1",
		}},
	}
	h := testAccessHandlerWithECS(&mockSessionStoreAccess{}, targets)
	body := sessionStartRequest{
		DeploymentName: "us-east-1",
		Target:         "eph-test-mc01",
		Reason:         "TEST-12345",
	}

	line := captureStdoutAccessEMF(func() {
		rr := doAccessRequest(h, "POST", "/api/v0/sessions/start", body, accessHeaders())
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
	if !strings.Contains(line, "eph-test-mc01") {
		t.Fatalf("expected EMF Cluster target id in stdout: %s", line)
	}
	if !strings.Contains(line, "access") {
		t.Fatalf("expected HandlerMode access in EMF: %s", line)
	}
}
