//go:build e2e

// Central profile: zoa targets lists SSM target rows for the deployment (vpc_id, target_type).

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa targets", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			var (
				deployment string
				targetID   string
			)

			BeforeEach(func() {
				skipUnlessSessionE2E()
				ctx, err := resolveSessionE2EContext(tgt.TargetType)
				Expect(err).NotTo(HaveOccurred())
				deployment = ctx.Deployment
				targetID = ctx.TargetID
			})

			It("lists targets for the deployment for this worker", Label("smoke"), func() {
				tgtOut, err := runZoaAccess("targets", deployment, "-o", "json")
				Expect(err).NotTo(HaveOccurred(), "zoa targets %q:\n%s", deployment, tgtOut)

				rows, err := parseSessionTargetsJSON(tgtOut)
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				Expect(rows).NotTo(BeEmpty(), "no targets for deployment %q", deployment)

				id, err := sessionTargetIDForType(rows, tgt.TargetType)
				Expect(err).NotTo(HaveOccurred(), "targets for type %q: %v", tgt.TargetType, rows)
				Expect(id).To(Equal(targetID))

				row := findSessionTargetRow(rows, targetID)
				Expect(row).NotTo(BeNil(), "target %q missing from targets list", targetID)
				Expect(row.VpcID).To(MatchRegexp(`^vpc-`),
					"target %q SSM row must include vpc_id (MC: mc-zoa-target-ssm pipeline; RC: zoa-lambda target registration)",
					targetID)
			})
		})
	}
})
