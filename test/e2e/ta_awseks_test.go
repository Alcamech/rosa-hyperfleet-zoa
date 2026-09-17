//go:build e2e

// TA: list_eks_clusters, describe_eks_cluster (aws-api scope)

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("list_eks_clusters", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			// Smoke test: Verifies aws-api scope works (STS AssumeRole, AWS SDK call).
			// Fast (~1s) and read-only.
			It("lists EKS clusters in the account", Label("smoke"), func() {
				list := runAction(tgt, "list_eks_clusters")
				out := outputMap(list)
				Expect(out).To(HaveKey("clusters"))
				Expect(out).To(HaveKey("count"))

				clusters, ok := out["clusters"].([]interface{})
				Expect(ok).To(BeTrue())
				Expect(clusters).NotTo(BeEmpty(), "expected at least the EKS cluster ZOA itself runs on")
			})
		})
	}
})

var _ = Describe("describe_eks_cluster", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("describes an existing cluster", func() {
				// Validate the TA round-trip only — not that unrelated EKS
				// clusters in a shared ephemeral account happen to be ACTIVE.
				var name string
				var detail map[string]interface{}
				Eventually(func() bool {
					name, detail = firstDescribableEKSCluster(tgt)
					return detail != nil
				}, "30s", "2s").Should(BeTrue(),
					"describe_eks_cluster should succeed for at least one listed cluster (override with E2E_EKS_CLUSTER_NAME)")

				Expect(detail["name"]).To(Equal(name))
				Expect(detail["status"]).NotTo(BeEmpty())
				Expect(detail["status"]).To(BeElementOf(
					"CREATING", "ACTIVE", "DELETING", "FAILED", "UPDATING", "PENDING",
				))
			})

			It("rejects describe without a name parameter", func() {
				out := runActionExpectFailure(tgt, "describe_eks_cluster")
				Expect(out).To(ContainSubstring("name"))
			})

			It("fails clearly for a nonexistent cluster", func() {
				out := runActionExpectFailure(tgt, "describe_eks_cluster", "--name", "e2e-nonexistent-cluster-zoa")
				Expect(out).NotTo(BeEmpty())
			})
		})
	}
})
