//go:build e2e

// Functional e2e: Access plane (central profile) — deployments, targets, boundary session lifecycle.
// First validation pass for operator session flow (--no-connect). Specs use Label("smoke") so
// consumer presubmits (make test-e2e-smoke) run them; they also run in make test-e2e / test-e2e-zoa.
// Skipped unless accessE2EEnabled() — TEMPORARY "eph" gate in access_helpers_test.go (int/stage CI).

package e2e

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/openshift-online/rosa-seaplane-zoa/pkg/reason"
)

var _ = Describe("zoa access validation", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			var (
				deployment   string
				accessTarget string
			)

			BeforeEach(func() {
				if !accessE2EEnabled() {
					Skip("Access smoke skipped: ZOA_DEPLOYMENT_NAME must contain eph until int/stage CI has central credentials (temporary gate)")
				}
				dep, err := resolveAccessDeployment()
				Expect(err).NotTo(HaveOccurred())
				Expect(dep).NotTo(BeEmpty())
				deployment = dep

				tgtOut, err := runZoaAccess("targets", deployment, "-o", "json")
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				rows, err := parseAccessTargetsJSON(tgtOut)
				Expect(err).NotTo(HaveOccurred(), tgtOut)
				accessTarget, err = accessTargetIDForType(rows, tgt.TargetType)
				Expect(err).NotTo(HaveOccurred(), "targets for %s: %v", tgt.TargetType, rows)
			})

			It("rejects session start without --reason", Label("smoke"), func() {
				out, err := runZoaAccessExpectFailure(
					"session", "start", deployment, accessTarget,
					"--no-connect",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(SatisfyAny(
					ContainSubstring("reason"),
					ContainSubstring("required"),
				))
			})

			It("rejects session start with invalid reason", Label("smoke"), func() {
				out, err := runZoaAccessExpectFailure(
					"session", "start", deployment, accessTarget,
					"--reason", "invalid-format",
					"--no-connect",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(SatisfyAny(
					ContainSubstring(reason.ExampleJiraIssue),
					ContainSubstring("not valid"),
				))
			})

			It("rejects session start for unknown target", Label("smoke"), func() {
				out, err := runZoaAccessExpectFailure(
					"session", "start", deployment, "e2e-nonexistent-target-zoa",
					"--reason", reasonValue,
					"--no-connect",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(out).To(SatisfyAny(
					ContainSubstring("target_not_found"),
					ContainSubstring("not found"),
				))
			})
		})
	}
})

var _ = Describe("zoa access", Serial, func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, Ordered, func() {
			var (
				deployment      string
				accessTarget    string
				compoundID      string
				rawID           string
				sessionOperator string
				accessReady     bool
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

				row := findAccessTargetRow(rows, accessTarget)
				Expect(row).NotTo(BeNil(), "target %q missing from targets list", accessTarget)
				Expect(row.VpcID).To(MatchRegexp(`^vpc-`),
					"target %q SSM row must include vpc_id (MC: mc-zoa-target-ssm pipeline; RC: zoa-lambda target registration)",
					accessTarget)
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
				Expect(strings.ToLower(started.Status)).To(Equal("active"),
					"--no-connect should provision the boundary task without ECS Exec")
				compoundID = started.SessionID
				rawID = started.RawID
				Expect(validateSessionStartResponse(started, deployment, accessTarget, rawID)).NotTo(HaveOccurred())
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
				Expect(validateActiveSessionRecord(*row, accessTarget, rawID, reasonValue)).NotTo(HaveOccurred())
				sessionOperator = row.Operator
			})

			It("includes the session in history with full audit fields", Label("smoke"), func() {
				Expect(rawID).NotTo(BeEmpty())
				Expect(sessionOperator).NotTo(BeEmpty(), "list active spec must run first")

				histOut, err := runZoaAccess(
					"session", "history", deployment,
					"--since", "1h",
					"-t", accessTarget,
					"-o", "json",
				)
				Expect(err).NotTo(HaveOccurred(), histOut)

				rows, err := parseSessionListJSON(histOut)
				Expect(err).NotTo(HaveOccurred(), histOut)
				row := findSessionForTarget(rows, rawID, accessTarget)
				Expect(row).NotTo(BeNil(),
					"session %q for target %q not in history: %v", rawID, accessTarget, rows)
				Expect(validateActiveSessionRecord(*row, accessTarget, rawID, reasonValue)).NotTo(HaveOccurred())
				Expect(row.Operator).To(Equal(sessionOperator),
					"history operator should match list for the same session")
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
				Expect(validateTerminatedSessionRecord(*row, accessTarget, rawID, reasonValue)).NotTo(HaveOccurred())
			})
		})
	}
})
