//go:build e2e_monitoring

package e2e_monitoring

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Recording rules from rosa-seaplane alerting-rules/templates/zoa.yaml (zoa-recording-rules group).
// Verifies Thanos Ruler has loaded each rule — same pattern as alerts_test.go.
var zoaRecordingRules = []string{
	"zoa:ta_success_rate",
	"zoa:ta_total_executions",
	"zoa:api_availability",
	"zoa:access_availability",
	"zoa:lambda_invocations_combined",
	"zoa:lambda_errors_combined",
	"zoa:lambda_throttles_combined",
	"zoa:lambda_error_rate",
	"zoa:lambda_throttle_rate",
	"zoa:reconciler_last_run",
	"zoa:gc_last_run",
	"zoa:reconciler_tick_count",
	"zoa:gc_tick_count",
	"zoa:reaper_last_run",
	"zoa:reaper_tick_count",
	"zoa:identity_bridge_hit_rate",
}

var _ = Describe("ZOA Recording Rules", func() {
	for _, rule := range zoaRecordingRules {
		rule := rule
		It("should have "+rule+" loaded in Thanos Ruler", Label("smoke"), func() {
			Eventually(func() bool {
				return hasRule(client, "record", rule)
			}, "5m", "15s").Should(BeTrue(),
				"Recording rule %s should be loaded in Thanos Ruler", rule)
		})
	}
})
