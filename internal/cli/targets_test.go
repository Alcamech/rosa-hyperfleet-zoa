package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/openshift-online/rosa-seaplane-zoa/internal/client"
	"github.com/openshift-online/rosa-seaplane-zoa/internal/output"
)

func TestTargets_WhenDeploymentMissing_ItShouldReturnHelpfulError(t *testing.T) {
	root := &cobra.Command{Use: "zoa"}
	root.AddCommand(newTargetsCommand(&GlobalOptions{}))
	root.SetArgs([]string{"targets"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when deployment is missing")
	}
	msg := err.Error()
	if !strings.Contains(msg, "deployment is required") {
		t.Fatalf("expected deployment hint, got: %s", msg)
	}
	if !strings.Contains(msg, "zoa deployments") {
		t.Fatalf("expected pointer to zoa deployments, got: %s", msg)
	}
}

func TestPrintTargetsTable_WhenDefault_ItShouldOmitInfrastructureColumns(t *testing.T) {
	items := []client.Target{{
		TargetID:          "eph-mc01",
		Region:            "us-east-1",
		TargetType:        "mc",
		Status:            "ready",
		AccountId:         "855246887846",
		VpcId:             "vpc-abc",
		SubnetIds:         "subnet-a,subnet-b",
		SecurityGroupId:   "sg-1",
		EcsClusterArn:     "arn:aws:ecs:us-east-1:123:cluster/c",
		TaskDefinitionArn: "arn:aws:ecs:us-east-1:123:task-definition/td:1",
		FunctionUrl:       "https://api.lambda-url.on.aws/",
	}}

	var buf bytes.Buffer
	opts := &GlobalOptions{OutputFormat: output.FormatTable}
	if err := printTargetsTable(&buf, opts, items); err != nil {
		t.Fatalf("printTargetsTable: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "VPC") || strings.Contains(out, "vpc-abc") || strings.Contains(out, "lambda-url") {
		t.Fatalf("default table should not show VPC/URLs, got:\n%s", out)
	}
	for _, want := range []string{"eph-mc01", "mc", "ready", "us-east-1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain %q, got:\n%s", want, out)
		}
	}
}

func TestPrintTargetsTable_WhenWide_ItShouldIncludeAllMetadata(t *testing.T) {
	items := []client.Target{{
		TargetID:          "eph-mc01",
		DeploymentName:    "us-east-1-eph",
		Region:            "us-east-1",
		TargetType:        "mc",
		Status:            "ready",
		AccountId:         "855246887846",
		VpcId:             "vpc-abc",
		SubnetIds:         "subnet-a,subnet-b",
		SecurityGroupId:   "sg-1",
		EcsClusterArn:     "arn:aws:ecs:us-east-1:123:cluster/c",
		TaskDefinitionArn: "arn:aws:ecs:us-east-1:123:task-definition/td:1",
		FunctionUrl:       "https://api.lambda-url.on.aws/",
	}}

	var buf bytes.Buffer
	opts := &GlobalOptions{OutputFormat: output.FormatWide}
	if err := printTargetsTable(&buf, opts, items); err != nil {
		t.Fatalf("printTargetsTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"SUBNETS", "subnet-a,subnet-b", "SECURITY GROUP", "sg-1",
		"ECS CLUSTER", "TASK DEFINITION", "FUNCTION URL", "https://api.lambda-url.on.aws/",
		"855246887846", "vpc-abc",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected wide output to contain %q, got:\n%s", want, out)
		}
	}
}
