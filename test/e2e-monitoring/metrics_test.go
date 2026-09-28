//go:build e2e_monitoring

package e2e_monitoring

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ZOA Metrics", func() {

	// Infrastructure metrics are always-on — they do not depend on TA
	// executions. nilToZero in YACE ensures error/throttle counters produce
	// 0-value series even when healthy.
	//
	// NOTE: GC metrics (aws_zoa_gclast_run_*, zoa:gc_last_run, zoa:gc_tick_count)
	// are excluded from E2E because GC runs every 5 min + YACE 600s period +
	// CW pipeline delay = up to 12 min before series appear in Thanos.
	// That blows the CI timeout budget for a signal that's redundant here:
	// the reconciler tick proves the same EMF → CW → YACE → Thanos path.
	//
	// GC *alerts* (ZOAGCStalled, ZOAGCErrors) are still tested in
	// alerts_test.go — those verify the rule definition is loaded in
	// Thanos Ruler, which succeeds regardless of whether the underlying
	// metric has data yet.
	Context("infrastructure metrics", func() {

		It("should have Lambda invocation metrics for ZOA functions", func() {
			query := `count(aws_lambda_invocations_sum{dimension_FunctionName=~".*-zoa-(api|worker)"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_lambda_invocations_sum for ZOA Lambda functions")
		})

		It("should have Lambda error metrics for ZOA functions", func() {
			query := `count(aws_lambda_errors_sum{dimension_FunctionName=~".*-zoa-(api|worker)"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_lambda_errors_sum (nilToZero) — feeds zoa:lambda_error_rate")
		})

		It("should have Lambda throttle metrics for ZOA functions", func() {
			query := `count(aws_lambda_throttles_sum{dimension_FunctionName=~".*-zoa-(api|worker)"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_lambda_throttles_sum (nilToZero) — feeds ZOALambdaThrottled alert")
		})

		It("should have SQS DLQ depth metrics", func() {
			query := `count(aws_sqs_approximate_number_of_messages_visible_maximum{dimension_QueueName=~".*-zoa-dlq"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected DLQ depth metric (nilToZero) — feeds ZOADLQGrowing alert")
		})

		It("should have reconciler tick metrics", func() {
			query := `count(aws_zoa_reconciler_last_run_maximum) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_zoa_reconciler_last_run_maximum (EMF ReconcilerLastRun)")
		})

		// NOTE: aws_dynamodb_throttled_requests_sum excluded — DynamoDB does
		// not publish ThrottledRequests until throttling occurs (nilToZero
		// does not help here). aws_zoa_gclast_run_maximum excluded — too slow.
	})

	// Execution metrics require TA runs to produce data. Smoke ZOA E2E
	// executes both sync and async TAs against RC and MC targets.
	Context("execution metrics", func() {

		It("should have execution count metrics", func() {
			query := `count(aws_zoa_execution_count_sum) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_zoa_execution_count_sum after TA executions")
		})

		It("should have execution metrics with Status dimension", func() {
			query := `count(aws_zoa_execution_count_sum{dimension_Status="succeeded"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected dimension_Status=succeeded on execution metrics")
		})

		It("should have HTTP request metrics", func() {
			query := `count(aws_zoa_http_request_count_sum) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_zoa_http_request_count_sum after API calls")
		})
	})

	// Cluster label validation: ZOA deploys per-VPC (RC + MC). MC metrics
	// traverse: MC Prometheus → SigV4 remote-write proxy → RC Thanos.
	// If the proxy breaks, MC alerts go blind silently.
	//
	// Raw metrics have cluster_type (from Prometheus externalLabels).
	// Recording rules aggregate by (cluster) which strips cluster_type,
	// so recording rule checks use cluster name pattern instead.
	Context("cluster labels", func() {

		It("should have EMF metrics from the regional cluster", func() {
			query := `count(aws_zoa_reconciler_last_run_maximum{cluster_type="regional-cluster"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected ZOA EMF metrics with cluster_type=regional-cluster")
		})

		It("should have EMF metrics from management clusters", func() {
			query := `count(aws_zoa_reconciler_last_run_maximum{cluster_type="management-cluster"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected ZOA EMF metrics with cluster_type=management-cluster")
		})

		It("should have Lambda metrics from the regional cluster", func() {
			query := `count(aws_lambda_invocations_sum{dimension_FunctionName=~".*-zoa-.*", cluster_type="regional-cluster"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected Lambda ZOA metrics with cluster_type=regional-cluster")
		})

		It("should have Lambda metrics from management clusters", func() {
			query := `count(aws_lambda_invocations_sum{dimension_FunctionName=~".*-zoa-.*", cluster_type="management-cluster"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected Lambda ZOA metrics with cluster_type=management-cluster")
		})
	})

	// Recording rule values: verify rules produce results for both RC and MC.
	// Rules aggregate by (cluster) which strips cluster_type, so we use
	// cluster name patterns: ".*-regional" for RC, ".*-mc.*" for MC.
	//
	// GC recording rules (zoa:gc_last_run, zoa:gc_tick_count) are excluded —
	// their input metrics take up to 12 min to appear (see infrastructure
	// metrics comment). The reconciler rules prove the same pipeline.
	// GC *alert* definitions are still verified in alerts_test.go.
	Context("recording rule values", func() {

		type check struct {
			rule         string
			clusterLabel string
			desc         string
		}

		rules := []check{
			{"zoa:reconciler_last_run", ".*-regional", "RC"},
			{"zoa:reconciler_last_run", ".*-mc.*", "MC"},
			{"zoa:reconciler_tick_count", ".*-regional", "RC"},
			{"zoa:reconciler_tick_count", ".*-mc.*", "MC"},
			{"zoa:lambda_error_rate", ".*-regional", "RC"},
			{"zoa:lambda_error_rate", ".*-mc.*", "MC"},
			{"zoa:ta_success_rate", ".*-regional", "RC"},
			{"zoa:ta_success_rate", ".*-mc.*", "MC"},
			{"zoa:ta_total_executions", ".*-regional", "RC"},
			{"zoa:ta_total_executions", ".*-mc.*", "MC"},
			{"zoa:api_availability", ".*-regional", "RC"},
			{"zoa:api_availability", ".*-mc.*", "MC"},
		}

		for _, tc := range rules {
			tc := tc
			It("should have "+tc.rule+" producing values for "+tc.desc, func() {
				query := `count(` + tc.rule + `{cluster=~"` + tc.clusterLabel + `"}) > 0`
				Eventually(func() bool {
					resp := thanosQuery(client, query)
					return resp.Status == "success" && len(resp.Data.Result) > 0
				}, "5m", "15s").Should(BeTrue(),
					"%s should produce values for %s (cluster=~%s)", tc.rule, tc.desc, tc.clusterLabel)
			})
		}
	})
})
