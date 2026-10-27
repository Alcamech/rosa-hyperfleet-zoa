//go:build e2e

// Package e2e is ZOA's deep Trusted Action validation suite. It drives the
// zoa CLI against real, already-provisioned RC and/or MC ZOA Lambda API URLs
// and exercises every registered Trusted Action (read/write, kube-api/aws-api,
// dry-run, error paths, and the audit trail).
//
// This suite does not provision any infrastructure itself. Point it at an
// existing environment via ZOA_RC_API_URL / ZOA_MC_API_URL and it can be
// re-run repeatedly against the same environment without re-provisioning.
//
// Run with: go test -tags e2e ./test/e2e/... -v
package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ZOA E2E Suite")
}

// This spec is registered unconditionally (i.e. it exists even when
// targets is empty) so a missing ZOA_RC_API_URL/ZOA_MC_API_URL produces a
// loud test failure instead of a silent "0 specs ran, SUCCESS" — Ginkgo
// skips BeforeSuite/AfterSuite entirely when there are zero specs to run,
// so relying on BeforeSuite alone for this check would be a false-positive
// footgun in CI.
var _ = Describe("suite configuration", func() {
	It("has at least one target configured", func() {
		Expect(targets).NotTo(BeEmpty(),
			"no target configured — export ZOA_RC_API_URL and/or ZOA_MC_API_URL before running this suite")
	})

	It("has ZOA_DEPLOYMENT_NAME set", func() {
		Expect(zoaDeploymentName()).NotTo(BeEmpty(),
			"ZOA_DEPLOYMENT_NAME is required — export it before running this suite")
	})
})

var _ = BeforeSuite(func() {
	for _, tgt := range targets {
		// `zoa version` deliberately never fails (it prints "Server: unreachable"
		// and returns nil so `zoa version` works offline) so it can't be used as
		// a health check. `zoa actions` makes a real authenticated call and
		// propagates connectivity/auth/profile errors as a non-zero exit —
		// exactly the class of failure (e.g. a missing AWS_PROFILE) this check
		// exists to catch fast, before every spec fails with the same error.
		out, err := runZoa(tgt, "actions")
		Expect(err).NotTo(HaveOccurred(),
			"pre-flight check failed for target %q (%s, AWS_PROFILE=%s) — is the URL reachable and is that profile authorized to invoke it?\n%s",
			tgt.Name, tgt.APIURL, tgt.AWSProfile, out)
		fmt.Printf("target %q ready: %s (AWS_PROFILE=%s)\n", tgt.Name, tgt.APIURL, tgt.AWSProfile)
	}
})
