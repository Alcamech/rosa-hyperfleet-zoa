//go:build e2e

// TA: delete_pod (kube-api scope, write) — deletes a real owned pod via ZOA.

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("delete_pod", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("deletes a real coredns pod via ZOA", func() {
				before := coreDNSPodNames(tgt)
				Expect(before).NotTo(BeEmpty(), "coredns pods not found (selector %q in %s)", coreDNSSelector, coreDNSNamespace)
				victim := before[0]

				// coredns pods are owned by a ReplicaSet, so delete_pod's
				// ownerReferences safety check passes.
				exec := runAction(tgt, "delete_pod", "--namespace", coreDNSNamespace, "--name", victim, "--force")
				Expect(exec["status"]).To(Equal("succeeded"))
				Expect(exec["action"]).To(Equal("delete_pod"), "this must be a real (non-dry-run) execution")
				Expect(outputMap(exec)).To(HaveKeyWithValue("status", "deleted"))
			})

			It("rejects a missing required parameter before touching the cluster", func() {
				out := runActionExpectFailure(tgt, "delete_pod", "--namespace", coreDNSNamespace)
				Expect(out).To(ContainSubstring("name"))
			})

			It("rejects a pod that does not exist", func() {
				out := runActionExpectFailure(tgt, "delete_pod",
					"--namespace", coreDNSNamespace, "--name", "e2e-nonexistent-pod-zoa", "--dry-run")
				Expect(out).To(SatisfyAny(
					ContainSubstring("not found"),
					ContainSubstring("could not find"),
				))
			})
		})
	}
})
