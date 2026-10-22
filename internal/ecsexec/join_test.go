package ecsexec

import "testing"

func TestJoinParams_WhenCommandEmpty_ItShouldUseDefault(t *testing.T) {
	p := JoinParams{
		Region:  "us-east-1",
		TaskARN: "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/abc123",
	}
	if p.Command != "" {
		t.Fatalf("expected empty Command before ConnectInteractive normalisation")
	}
	if defaultExecCommand != "/bin/bash" {
		t.Errorf("defaultExecCommand = %q, want /bin/bash", defaultExecCommand)
	}
}

func TestJoinParams_WhenContainerNameEmpty_ItShouldDefaultToZoaBoundary(t *testing.T) {
	p := JoinParams{
		Region:  "us-east-1",
		TaskARN: "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/abc123",
	}
	if p.ContainerName != "" {
		t.Fatalf("expected empty ContainerName before ConnectInteractive normalisation")
	}
}

func TestJoinParams_WhenTaskARNParsed_ItShouldExtractClusterAndTask(t *testing.T) {
	p := JoinParams{
		TaskARN: "arn:aws:ecs:us-east-1:123456789012:task/my-cluster/abc123",
	}
	cluster, taskID, err := ClusterAndTaskFromTaskARN(p.TaskARN)
	if err != nil {
		t.Fatal(err)
	}
	if cluster != "my-cluster" {
		t.Errorf("cluster = %q", cluster)
	}
	if taskID != "abc123" {
		t.Errorf("taskID = %q", taskID)
	}
}

func TestJoinParams_WhenClusterOverrideSet_ItShouldPreferOverride(t *testing.T) {
	p := JoinParams{
		Cluster: "arn:aws:ecs:us-east-1:123456789012:cluster/override-cluster",
		TaskARN: "arn:aws:ecs:us-east-1:123456789012:task/task-cluster/abc123",
	}
	cluster := ClusterNameFromARN(p.Cluster)
	if cluster != "override-cluster" {
		t.Errorf("cluster = %q, want override-cluster", cluster)
	}
}

func TestJoinParams_WhenOnExecSessionSet_ItShouldBeCallable(t *testing.T) {
	called := false
	p := JoinParams{
		OnExecSession: func(execSessionID string) error {
			called = true
			if execSessionID != "sess-test" {
				t.Errorf("execSessionID = %q", execSessionID)
			}
			return nil
		},
	}
	if err := p.OnExecSession("sess-test"); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("OnExecSession was not called")
	}
}
