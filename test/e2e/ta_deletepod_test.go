//go:build e2e

// TA: delete_pod (kube-api scope, write) — deletes a real owned pod via ZOA.

package e2e

import (
	"sort"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa run delete_pod", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			BeforeEach(func() {
				skipUnlessLiveAction(tgt, "delete_pod")
			})

			It("deletes a real coredns pod via ZOA", func() {
				// E2E_COREDNS_* env vars (helpers_test.go) pick the workload; list+delete
				// go through TAs (get_resource, delete_pod). Retries tolerate rollout churn.
				var exec map[string]interface{}
				Eventually(func(g Gomega) {
					names, err := deletePodTestEligiblePodNames(tgt, coreDNSNamespace, coreDNSSelector)
					g.Expect(err).NotTo(HaveOccurred())
					g.Expect(names).NotTo(BeEmpty(),
						"no eligible pods (selector %q in %s)", coreDNSSelector, coreDNSNamespace)
					sort.Strings(names)
					exec, err = runActionResult(tgt, "delete_pod",
						"--namespace", coreDNSNamespace,
						"--name", names[0],
						"--force",
					)
					g.Expect(err).NotTo(HaveOccurred())
				}, 60*time.Second, 5*time.Second).Should(Succeed())

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

// deletePodTestEligiblePodNames lists pods for the live delete_pod spec via get_resource,
// skipping Terminating/Completed Table rows.
func deletePodTestEligiblePodNames(tgt target, namespace, selector string) ([]string, error) {
	args := []string{"--resource", "pods", "--namespace", namespace}
	if strings.TrimSpace(selector) != "" {
		args = append(args, "--selector", selector)
	}
	listExec, err := runActionResult(tgt, "get_resource", args...)
	if err != nil {
		return nil, err
	}
	rows, ok := listExec["output"].([]interface{})
	if !ok {
		return nil, nil
	}
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		row, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if status, ok := caseInsensitiveField(row, "status"); ok {
			lower := strings.ToLower(strings.TrimSpace(status))
			if strings.Contains(lower, "terminating") || strings.Contains(lower, "completed") {
				continue
			}
		}
		if name, ok := caseInsensitiveField(row, "name"); ok {
			names = append(names, name)
		}
	}
	return names, nil
}
