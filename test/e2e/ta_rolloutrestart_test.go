//go:build e2e

// TA: rollout_restart (kube-api scope, write) — restarts workloads via ZOA.

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("rollout_restart", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			// Smoke-level coverage stays --dry-run: consumer repos
			// (rosa-hyperfleet, rosa-hyperfleet-api) run this on every
			// on-demand-e2e/nightly-ephemeral job via `make
			// test-e2e-smoke`, so it must never mutate a live cluster.
			It("dry-run: validates a real Deployment target without mutating it", Label("smoke"), func() {
				exec := runAction(tgt, "rollout_restart",
					"--resource", "deployment", "--namespace", coreDNSNamespace, "--name", coreDNSName, "--dry-run")
				Expect(exec["status"]).To(Equal("succeeded"))
				Expect(exec["action"]).To(Equal("get_resource"), "a dry-run must execute get_resource, never the real mutation")
			})

			It("restarts the real coredns Deployment via ZOA", func() {
				// --force bypasses the write cooldown so this suite can be
				// re-run in quick succession without waiting out
				// rollout_restart's 300s cooldown.
				exec := runAction(tgt, "rollout_restart",
					"--resource", "deployment", "--namespace", coreDNSNamespace, "--name", coreDNSName, "--force")
				Expect(exec["status"]).To(Equal("succeeded"))
				Expect(exec["action"]).To(Equal("rollout_restart"), "this must be a real (non-dry-run) execution")

				out := outputMap(exec)
				Expect(out["status"]).To(BeElementOf("restarted", "restart-initiated"))
			})

			It("rejects an unsupported resource type before touching the cluster", func() {
				// --force bypasses cooldown so we can test the actual param validation
				out := runActionExpectFailure(tgt, "rollout_restart",
					"--resource", "cronjob", "--namespace", coreDNSNamespace, "--name", "whatever", "--force")
				Expect(out).To(ContainSubstring("unsupported resource"))
			})

			It("rejects a workload that does not exist", func() {
				out := runActionExpectFailure(tgt, "rollout_restart",
					"--resource", "deployment", "--namespace", coreDNSNamespace, "--name", "e2e-nonexistent-deployment-zoa", "--dry-run")
				Expect(out).To(SatisfyAny(
					ContainSubstring("not found"),
					ContainSubstring("could not find"),
				))
			})
		})
	}
})
