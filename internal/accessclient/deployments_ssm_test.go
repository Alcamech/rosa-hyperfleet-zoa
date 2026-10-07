package accessclient

import (
	"testing"
)

func TestDeploymentsSSMRegionsToTry_WhenExplicitSet_ItShouldUseOnlyThatRegion(t *testing.T) {
	got := DeploymentsSSMRegionsToTry("eu-west-1")
	if len(got) != 1 || got[0] != "eu-west-1" {
		t.Fatalf("got %v", got)
	}
}

func TestDeploymentsSSMRegionsToTry_WhenEmpty_ItShouldFailoverPrimaryThenReplica(t *testing.T) {
	got := DeploymentsSSMRegionsToTry("")
	if len(got) != 2 || got[0] != DefaultDeploymentsSSMRegion || got[1] != ReplicaDeploymentsSSMRegion {
		t.Fatalf("got %v", got)
	}
}
