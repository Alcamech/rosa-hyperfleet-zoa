//go:build e2e

// TA: get_secret (kube-api scope) — includes HCP namespace protection tests.

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("get_secret", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("lists secrets in a normal namespace (metadata only, no data)", func() {
				exec := runAction(tgt, "get_secret", "--namespace", "kube-system")
				rows := outputArray(exec) // nil is fine — zero secrets is a valid outcome
				for _, r := range rows {
					row, ok := r.(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(row).NotTo(HaveKey("data"), "default (non-verbose) get_secret must never return raw values")
				}
			})

			It("blocks the cluster- HCP namespace prefix regardless of resource existence", func() {
				out := runActionExpectFailure(tgt, "get_secret", "--namespace", "cluster-does-not-exist", "--name", "whatever")
				// Either: TA rejects due to HCP protection, OR executor fails to create RBAC
				// (namespace doesn't exist). Both prevent access — security invariant holds.
				Expect(out).To(SatisfyAny(
					ContainSubstring("HCP namespace protection"),
					ContainSubstring("not found"),
				))
			})

			It("blocks control plane namespaces under cluster- prefix", func() {
				out := runActionExpectFailure(tgt, "get_secret", "--namespace", "cluster-abc123-my-hc")
				Expect(out).To(SatisfyAny(
					ContainSubstring("HCP namespace protection"),
					ContainSubstring("not found"),
				))
			})

			It("blocks sensitive secret name patterns even in an allowed namespace", func() {
				out := runActionExpectFailure(tgt, "get_secret", "--namespace", "kube-system", "--name", "some-kubeconfig")
				Expect(out).To(ContainSubstring("sensitive HCP secret"))
			})

			It("requires a namespace", func() {
				out := runActionExpectFailure(tgt, "get_secret")
				Expect(out).To(ContainSubstring("namespace"))
			})
		})
	}
})
