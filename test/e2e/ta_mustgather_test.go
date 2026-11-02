//go:build e2e

// TA: must_gather — platform dumps on RC (gather=rc) and MC (gather=mc).

package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa run must_gather", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			BeforeEach(func() {
				skipUnlessLiveAction(tgt, "must_gather")
			})

			It("rejects missing gather parameter", func() {
				out := runActionExpectFailure(tgt, "must_gather")
				// Missing CLI flag → API validateParams before TA Validate.
				Expect(out).To(SatisfyAny(
					ContainSubstring(`required parameter "gather" is missing`),
					ContainSubstring(`parameter "gather" is required`),
				))
			})

			It("rejects gather for the wrong deployment target", func() {
				wrongGather := "mc"
				if tgt.TargetType == "mc" {
					wrongGather = "rc"
				}
				out := runActionExpectFailure(tgt, "must_gather",
					"--gather", wrongGather,
				)
				Expect(out).To(ContainSubstring("is not allowed on " + tgt.TargetType + " ZOA endpoint"))
			})

			if tgt.TargetType == "mc" {
				It("rejects gather=hcp without cluster_id", func() {
					out := runActionExpectFailure(tgt, "must_gather",
						"--gather", "hcp",
					)
					Expect(out).To(ContainSubstring("cluster_id is required when gather includes hcp"))
				})
			}

			It("collects a platform tarball for this deployment target", func() {
				Expect(tgt.TargetType).To(SatisfyAny(Equal("rc"), Equal("mc")))

				exec := runAction(tgt, "must_gather",
					"--gather", tgt.TargetType,
					"--wait",
					"--wait-timeout", "20m",
					"--wait-poll-interval", "30s",
				)
				id, ok := exec["id"].(string)
				Expect(ok).To(BeTrue(), "expected execution id in response: %v", exec)

				tmpDir := GinkgoT().TempDir()
				tarFile := filepath.Join(tmpDir, "must-gather.tar.gz")
				Expect(downloadExecutionOutput(tgt, id, tarFile)).To(Succeed())

				marker := mustGatherPlatformNamespace(tgt.TargetType)
				Expect(marker).NotTo(BeEmpty())

				found, err := tarballContainsPrefix(tarFile, marker)
				Expect(err).NotTo(HaveOccurred())
				Expect(found).To(BeTrue(),
					"expected %q in must_gather tarball for %s gather=%s (platform namespace missing — is the component deployed?)",
					marker, tgt.Name, tgt.TargetType)

				opposite := mustGatherOppositePlatformPrefix(tgt.TargetType)
				foundOpposite, err := tarballContainsPrefix(tarFile, opposite+"namespaces/")
				Expect(err).NotTo(HaveOccurred())
				Expect(foundOpposite).To(BeFalse(),
					"did not expect opposite deployment tree %q in gather=%s tarball", opposite, tgt.TargetType)
			})
		})
	}
})
