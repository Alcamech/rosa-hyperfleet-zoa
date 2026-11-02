//go:build e2e

// Central profile: zoa deployments lists SSM-registered deployment names for this environment.

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa deployments", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			var deployment string

			BeforeEach(func() {
				skipUnlessSessionE2E()
				ctx, err := resolveSessionE2EContext(tgt.TargetType)
				Expect(err).NotTo(HaveOccurred())
				deployment = ctx.Deployment
			})

			It("lists central deployments including this environment", Label("smoke"), func() {
				depOut, err := runZoaAccess("deployments", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), "zoa deployments (profile %s):\n%s", centralProfile, depOut)
				Expect(depOut).To(ContainSubstring(deployment),
					"expected deployment %q in central SSM list", deployment)
			})
		})
	}
})
