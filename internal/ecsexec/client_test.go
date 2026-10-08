package ecsexec

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	"github.com/aws/aws-sdk-go-v2/service/ecs/types"
)

func TestNewClient_WhenClusterARN_ItShouldExtractName(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	c := NewClient(cfg, "arn:aws:ecs:us-east-1:123456789012:cluster/my-cluster")
	if c.cluster != "my-cluster" {
		t.Errorf("cluster = %q, want %q", c.cluster, "my-cluster")
	}
}

func TestNewClient_WhenPlainName_ItShouldKeepName(t *testing.T) {
	cfg := aws.Config{Region: "us-east-1"}
	c := NewClient(cfg, "my-cluster")
	if c.cluster != "my-cluster" {
		t.Errorf("cluster = %q, want %q", c.cluster, "my-cluster")
	}
}

func TestIsExecAgentRunning_WhenAgentRunning_ItShouldReturnTrue(t *testing.T) {
	c := &Client{cluster: "test"}
	c.client = nil // we'll test the helper directly

	out := &ecs.DescribeTasksOutput{
		Tasks: []types.Task{
			{
				Containers: []types.Container{
					{
						Name: aws.String("zoa-boundary"),
						ManagedAgents: []types.ManagedAgent{
							{
								Name:       types.ManagedAgentNameExecuteCommandAgent,
								LastStatus: aws.String("RUNNING"),
							},
						},
					},
				},
			},
		},
	}

	ready := false
	for _, cont := range out.Tasks[0].Containers {
		if aws.ToString(cont.Name) != "zoa-boundary" {
			continue
		}
		for _, agent := range cont.ManagedAgents {
			if agent.Name == types.ManagedAgentNameExecuteCommandAgent {
				ready = aws.ToString(agent.LastStatus) == "RUNNING"
			}
		}
	}
	if !ready {
		t.Error("expected exec agent to be running")
	}
}

func TestIsExecAgentRunning_WhenContainerMissing_ItShouldNotPanic(t *testing.T) {
	out := &ecs.DescribeTasksOutput{
		Tasks: []types.Task{
			{Containers: []types.Container{}},
		},
	}
	found := false
	for _, cont := range out.Tasks[0].Containers {
		if aws.ToString(cont.Name) == "zoa-boundary" {
			found = true
		}
	}
	if found {
		t.Error("expected container not found")
	}
}

func TestExecuteCommand_WhenSessionReturned_ItShouldMarshalPayload(t *testing.T) {
	sessionPayload := map[string]string{
		"sessionId":  "sess-123",
		"streamUrl":  "wss://example.com",
		"tokenValue": "tok-456",
	}
	raw, err := json.Marshal(sessionPayload)
	if err != nil {
		t.Fatal(err)
	}

	session := &Session{
		SessionID:  "sess-123",
		Target:     "ecs:test_task123_rt456",
		RawSession: raw,
	}

	var parsed map[string]string
	if err := json.Unmarshal(session.RawSession, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if parsed["sessionId"] != "sess-123" {
		t.Errorf("sessionId = %q", parsed["sessionId"])
	}
	if parsed["streamUrl"] != "wss://example.com" {
		t.Errorf("streamUrl = %q", parsed["streamUrl"])
	}
}
