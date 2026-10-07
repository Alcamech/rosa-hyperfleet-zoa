package accessclient

import (
	"strings"
	"testing"
)

func TestParseDeploymentSSMParameter_WhenValidJSON_ItShouldDecode(t *testing.T) {
	raw := `{
		"deployment_name": "us-east-1",
		"access_url": "https://example.lambda-url.us-east-1.on.aws/",
		"invoker_role_arn": "arn:aws:iam::123456789012:role/invoker",
		"region": "us-east-1",
		"account_id": "123456789012"
	}`
	dep, err := parseDeploymentSSMParameter("us-east-1", raw)
	if err != nil {
		t.Fatalf("parseDeploymentSSMParameter: %v", err)
	}
	if dep.AccessURL != "https://example.lambda-url.us-east-1.on.aws/" {
		t.Fatalf("access_url: %q", dep.AccessURL)
	}
}

func TestParseDeploymentSSMParameter_WhenInvalidJSON_ItShouldError(t *testing.T) {
	_, err := parseDeploymentSSMParameter("us-east-1", "{")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "parsing SSM parameter") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDeploymentSSMParameter_WhenMissingAccessURL_ItShouldError(t *testing.T) {
	raw := `{"deployment_name":"us-east-1","invoker_role_arn":"arn:aws:iam::123456789012:role/invoker"}`
	_, err := parseDeploymentSSMParameter("us-east-1", raw)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no access_url") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDeploymentSSMParameter_WhenMissingInvokerRole_ItShouldError(t *testing.T) {
	raw := `{"deployment_name":"us-east-1","access_url":"https://example.on.aws/"}`
	_, err := parseDeploymentSSMParameter("us-east-1", raw)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "no invoker_role_arn") {
		t.Fatalf("unexpected error: %v", err)
	}
}
