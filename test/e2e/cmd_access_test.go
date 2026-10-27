//go:build e2e

// CLI: Access plane (central profile) — deployments, targets, boundary session lifecycle.
// One Describe per RC/MC worker (parallel ginkgo processes). Access specs are skipped unless
// accessE2EEnabled() — see TEMPORARY "eph" gate in access_helpers_test.go (int/stage CI).

package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa access", Serial, func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, Ordered, func() {
			var (
				deployment   string
				accessTarget string
				compoundID   string
				rawID        string
				accessReady  bool
			)

			BeforeAll(func() {
				if !accessE2EEnabled() {
					return
				}
				dep, err := resolveAccessDeployment()
				Expect(err).NotTo(HaveOccurred(), "ZOA_DEPLOYMENT_NAME (central profile %s)", centralProfile)
				Expect(dep).NotTo(BeEmpty())
				deployment = dep

				tgtOut, err := runZoaAccess("targets", deployment, "-o", "json")
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				rows, err := parseAccessTargetsJSON(tgtOut)
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				accessTarget, err = accessTargetIDForType(rows, tgt.TargetType)
				Expect(err).NotTo(HaveOccurred(), "targets for %s: %v", tgt.TargetType, rows)
				accessReady = true
			})

			BeforeEach(func() {
				if !accessE2EEnabled() {
					Skip("Access smoke skipped: ZOA_DEPLOYMENT_NAME must contain eph until int/stage CI has central credentials (temporary gate)")
				}
				if !accessReady {
					Skip("Access deployment not resolved")
				}
			})

			It("lists central deployments including this environment", Label("smoke"), func() {
				depOut, err := runZoaAccess("deployments", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), "zoa deployments (profile %s):\n%s", centralProfile, depOut)
				Expect(depOut).To(ContainSubstring(deployment),
					"expected deployment %q in central SSM list", deployment)
			})

			It("lists targets for the deployment for this worker", Label("smoke"), func() {
				tgtOut, err := runZoaAccess("targets", deployment, "-o", "json")
				Expect(err).NotTo(HaveOccurred(), "zoa targets %q:\n%s", deployment, tgtOut)

				rows, err := parseAccessTargetsJSON(tgtOut)
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				Expect(rows).NotTo(BeEmpty(), "no targets for deployment %q", deployment)

				id, err := accessTargetIDForType(rows, tgt.TargetType)
				Expect(err).NotTo(HaveOccurred(), "targets for type %q: %v", tgt.TargetType, rows)
				Expect(id).To(Equal(accessTarget))
			})

			It("starts a boundary session without ECS Exec", Label("smoke"), func() {
				startOut, err := runZoaAccess(
					"session", "start", deployment, accessTarget,
					"--reason", reasonValue,
					"--no-connect", "-o", "json",
				)
				Expect(err).NotTo(HaveOccurred(), "zoa session start %q %q:\n%s", deployment, accessTarget, startOut)

				started, err := parseSessionStartJSON(startOut)
				Expect(err).NotTo(HaveOccurred(), startOut)
				Expect(started.SessionID).NotTo(BeEmpty())
				Expect(started.RawID).NotTo(BeEmpty())
				Expect(strings.EqualFold(started.Target, accessTarget)).To(BeTrue())
				Expect(strings.ToLower(started.Status)).To(Equal("active"),
					"--no-connect should provision the boundary task without ECS Exec")
				compoundID = started.SessionID
				rawID = started.RawID
			})

			It("lists the session for this target after start", Label("smoke"), func() {
				Expect(rawID).NotTo(BeEmpty(), "session start spec must run first in Ordered container")

				listOut, err := runZoaAccess(
					"session", "list", deployment,
					"-t", accessTarget,
					"--status", "active",
					"-o", "json",
				)
				Expect(err).NotTo(HaveOccurred(), listOut)

				rows, err := parseSessionListJSON(listOut)
				Expect(err).NotTo(HaveOccurred(), listOut)
				row := findSessionForTarget(rows, rawID, accessTarget)
				Expect(row).NotTo(BeNil(),
					"session %q for target %q not in scope=mine list (same central profile as start): %v",
					rawID, accessTarget, rows)
				Expect(strings.ToLower(row.Status)).To(Equal("active"))
			})

			It("includes the session in history for this target", Label("smoke"), func() {
				Expect(rawID).NotTo(BeEmpty())

				histOut, err := runZoaAccess(
					"session", "history", deployment,
					"--since", "1h",
					"-t", accessTarget,
					"-o", "json",
				)
				Expect(err).NotTo(HaveOccurred(), histOut)
				Expect(histOut).To(ContainSubstring(rawID))
			})

			It("terminates the session", Label("smoke"), func() {
				Expect(compoundID).NotTo(BeEmpty())

				termOut, err := runZoaAccess("session", "terminate", compoundID)
				Expect(err).NotTo(HaveOccurred(), termOut)
			})

			It("lists the session as terminated for this target", Label("smoke"), func() {
				Expect(rawID).NotTo(BeEmpty())

				listOut, err := runZoaAccess(
					"session", "list", deployment,
					"-t", accessTarget,
					"--status", "terminated",
					"-o", "json",
				)
				Expect(err).NotTo(HaveOccurred(), listOut)

				rows, err := parseSessionListJSON(listOut)
				Expect(err).NotTo(HaveOccurred(), listOut)
				row := findSessionForTarget(rows, rawID, accessTarget)
				Expect(row).NotTo(BeNil(), "terminated session for target %q not in list: %v", accessTarget, rows)
				Expect(strings.ToLower(row.Status)).To(Equal("terminated"))
			})
		})
	}
})
