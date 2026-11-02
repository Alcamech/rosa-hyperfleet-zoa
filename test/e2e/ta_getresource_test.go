//go:build e2e

// TA: get_resource (kube-api scope) — generic K8s resource reader.

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa run get_resource", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			BeforeEach(func() {
				skipUnlessLiveAction(tgt, "get_resource")
			})

			It("lists cluster-scoped resources (nodes)", Label("smoke"), func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				rows := outputArray(exec)
				Expect(rows).NotTo(BeEmpty(), "expected at least one node")
			})

			It("lists namespaced resources (pods in kube-system)", func() {
				exec := runAction(tgt, "get_resource", "--resource", "pods", "--namespace", "kube-system")
				rows := outputArray(exec)
				Expect(rows).NotTo(BeEmpty(), "expected kube-system to have pods")

				row, ok := rows[0].(map[string]interface{})
				Expect(ok).To(BeTrue())
				_, hasName := caseInsensitiveField(row, "name")
				Expect(hasName).To(BeTrue(), "expected a name column in table output: %v", row)
			})

			It("gets a single resource by name", func() {
				list := runAction(tgt, "get_resource", "--resource", "pods", "--namespace", "kube-system")
				rows := outputArray(list)
				Expect(rows).NotTo(BeEmpty())
				row, ok := rows[0].(map[string]interface{})
				Expect(ok).To(BeTrue())
				name, ok := caseInsensitiveField(row, "name")
				Expect(ok).To(BeTrue())

				// A --name lookup always resolves to exactly one row, so
				// get_resource returns it as a single flat object rather
				// than a one-element array (see getresource.go's getTable).
				exec := runAction(tgt, "get_resource", "--resource", "pods", "--namespace", "kube-system", "--name", name)
				single := outputMap(exec)
				gotName, ok := caseInsensitiveField(single, "name")
				Expect(ok).To(BeTrue())
				Expect(gotName).To(Equal(name))
			})

			It("rejects secrets — must use the dedicated get_secret action", func() {
				out := runActionExpectFailure(tgt, "get_resource", "--resource", "secrets", "--namespace", "kube-system")
				Expect(out).To(ContainSubstring("get_secret"))
			})
		})
	}
})
