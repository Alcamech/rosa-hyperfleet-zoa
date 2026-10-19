package api

import (
	"encoding/json"
	"testing"

	"github.com/openshift-online/rosa-seaplane-zoa/pkg/actions"
)

// TestCreateResponse_WhenSerialized_ItShouldIncludeAllCLIExpectedFields verifies
// that the dispatch response JSON includes all fields the CLI expects.
// This test would have caught the bug where action/scope/type were missing.
func TestCreateResponse_WhenSerialized_ItShouldIncludeAllCLIExpectedFields(t *testing.T) {
	durationMs := int64(123)
	resp := createResponse{
		ID:              "test-id",
		Action:          "get_resource",
		RequestedAction: "rollout_restart",
		TargetCluster:   "test-cluster",
		Operator:        "test-operator",
		Status:          "succeeded",
		ExecutionMode:   "sync",
		Scope:           "kube-api",
		Type:            "read",
		DryRun:          true,
		Force:           true,
		DurationMs:      &durationMs,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	// Unmarshal into a generic map to verify field names
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// These are the fields the CLI expects (from internal/client/types.go Execution struct)
	requiredFields := []string{
		"id",
		"action", // NOT "executed_action" - CLI expects "action"
		"target_cluster",
		"operator",
		"status",
		"execution_mode",
		"scope",
		"type",
		"dry_run",
		"force",
	}

	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("response JSON missing required field %q (CLI will see empty value)", field)
		}
	}

	// Verify specific values
	if result["action"] != "get_resource" {
		t.Errorf("expected action=%q, got %v", "get_resource", result["action"])
	}
	if result["operator"] != "test-operator" {
		t.Errorf("expected operator=%q, got %v", "test-operator", result["operator"])
	}
	if result["requested_action"] != "rollout_restart" {
		t.Errorf("expected requested_action=%q, got %v", "rollout_restart", result["requested_action"])
	}
	if result["scope"] != "kube-api" {
		t.Errorf("expected scope=%q, got %v", "kube-api", result["scope"])
	}
	if result["type"] != "read" {
		t.Errorf("expected type=%q, got %v", "read", result["type"])
	}
}

func TestValidateParams_WhenAllRequiredPresent_ItShouldSucceed(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: []actions.ParameterDef{
			{Name: "namespace", Required: true},
			{Name: "pod_name", Required: true},
			{Name: "verbose", Required: false},
		},
	}

	params := map[string]string{
		"namespace": "default",
		"pod_name":  "my-pod",
	}

	if err := validateParams(meta, params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateParams_WhenRequiredMissing_ItShouldReturnError(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: []actions.ParameterDef{
			{Name: "namespace", Required: true},
			{Name: "pod_name", Required: true},
		},
	}

	params := map[string]string{
		"namespace": "default",
	}

	err := validateParams(meta, params)
	if err == nil {
		t.Fatal("expected error for missing required param")
	}
	expected := `required parameter "pod_name" is missing`
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateParams_WhenUnknownParam_ItShouldReturnError(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: []actions.ParameterDef{
			{Name: "namespace", Required: true},
		},
	}

	params := map[string]string{
		"namespace": "default",
		"unknown":   "value",
	}

	err := validateParams(meta, params)
	if err == nil {
		t.Fatal("expected error for unknown param")
	}
	expected := `unknown parameter "unknown"`
	if err.Error() != expected {
		t.Fatalf("expected error %q, got %q", expected, err.Error())
	}
}

func TestValidateParams_WhenNoParams_ItShouldSucceed(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: []actions.ParameterDef{
			{Name: "verbose", Required: false, Default: "false"},
		},
	}

	params := map[string]string{}

	if err := validateParams(meta, params); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateParams_WhenNilParams_ItShouldCheckRequired(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: []actions.ParameterDef{
			{Name: "namespace", Required: true},
		},
	}

	err := validateParams(meta, nil)
	if err == nil {
		t.Fatal("expected error for nil params with required field")
	}
}

func TestValidateParams_WhenEmptyDefinition_ItShouldRejectAnyParams(t *testing.T) {
	meta := actions.ActionMetadata{
		Parameters: nil,
	}

	params := map[string]string{
		"unexpected": "value",
	}

	err := validateParams(meta, params)
	if err == nil {
		t.Fatal("expected error for params when no params defined")
	}
}
