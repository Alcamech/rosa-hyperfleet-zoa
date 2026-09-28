//go:build e2e_monitoring

package e2e_monitoring

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Alerting rules defined in rosa-hyperfleet alerting-rules/templates/zoa.yaml.
// All alerts are tested here, including GC alerts (ZOAGCStalled, ZOAGCErrors).
// Unlike raw/recording-rule metric tests, alert definition tests only check
// that the rule is loaded in Thanos Ruler — they don't need the underlying
// metric to have data yet, so GC's slow propagation delay is not an issue.
var zoaAlertingRules = []string{
	"ZOAExecutionsTotalFailure",
	"ZOAReconcilerStalled",
	"ZOADLQGrowing",
	"ZOADynamoDBThrottled",
	"ZOALambdaErrorRateHigh",
	"ZOAApiAvailabilityBelowSLO",
	"ZOATASuccessRateBelowSLO",
	"ZOALambdaThrottled",
	"ZOAGCStalled",
	"ZOAGCErrors",
	"ZOACircuitBreakerFlapping",
}

var _ = Describe("ZOA Alerting Rules", func() {
	for _, alert := range zoaAlertingRules {
		alert := alert // capture
		It("should have "+alert+" loaded in Thanos Ruler", func() {
			Eventually(func() bool {
				return hasRule(client, "alert", alert)
			}, "5m", "15s").Should(BeTrue(),
				"Alert %s should be loaded in Thanos Ruler", alert)
		})
	}
})
