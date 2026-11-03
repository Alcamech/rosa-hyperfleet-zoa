//go:build e2e_monitoring

package e2e_monitoring

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ZOA Metrics", func() {

	// Infrastructure metrics are always-on — they do not depend on TA
	// executions. nilToZero in YACE ensures error/throttle counters produce
	// 0-value series even when healthy.
	//
	// NOTE: GC and reaper EMF (aws_zoa_gclast_run_*, aws_zoa_reaper_last_run_*,
	// and recording rules zoa:gc_last_run, zoa:gc_tick_count, zoa:reaper_last_run,
	// zoa:reaper_tick_count) are excluded from value checks because both workers
	// tick every ~5 min + YACE 600s period + CW pipeline delay = up to ~12 min
	// before series appear in Thanos. That blows the CI timeout budget; reconciler
	// tick proves the same EMF → CW → YACE → Thanos path.
	//
	// GC/reaper *alerts* (ZOAGCStalled, ZOAGCErrors, ZOAReaperStalled,
	// ZOAReaperErrors) and recording rule *definitions* are still tested in
	// alerts_test.go / recording_rules_test.go (loaded in Thanos Ruler only).
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
			query := `count(aws_lambda_errors_sum{dimension_FunctionName=~".*-zoa-(api|worker|access)"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_lambda_errors_sum (nilToZero) for api/worker/access — feeds zoa:lambda_* recording rules")
		})

		It("should have Lambda throttle metrics for ZOA functions", func() {
			query := `count(aws_lambda_throttles_sum{dimension_FunctionName=~".*-zoa-(api|worker|access)"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_lambda_throttles_sum (nilToZero) — feeds zoa:lambda_throttle_rate")
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

		It("should have ECS boundary TaskCount metrics from Container Insights", func() {
			query := `count(aws_ecs_containerinsights_task_count_maximum{dimension_ClusterName=~".*-zoa-boundary"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected Container Insights TaskCount for *-zoa-boundary (nilToZero) — boundary running tasks panel")
		})

		// NOTE: aws_dynamodb_throttled_requests_sum excluded — DynamoDB does
		// not publish ThrottledRequests until throttling occurs (nilToZero
		// does not help here).
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

		It("should have Access HTTP request metrics after session smoke", func() {
			query := `count(aws_zoa_http_request_count_sum{dimension_HandlerMode="access"}) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected Access plane HTTP metrics (session start/list/terminate in smoke e2e)")
		})

		It("should have identity bridge lookup metrics", func() {
			query := `count(aws_zoa_identity_bridge_lookup_sum) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected aws_zoa_identity_bridge_lookup_sum (hit/miss/error/skipped) after API TA traffic")
		})
	})

	// Cluster label validation: ZOA deploys per-VPC (RC + MC). MC metrics
	// traverse: MC Prometheus → SigV4 remote-write proxy → RC Thanos.
	// If the proxy breaks, MC alerts go blind silently.
	//
	// RC/MC via cluster_type on externalLabels (not cluster name patterns).
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

	// Access availability uses EMF cluster labels (target id or deployment name), not
	// only Lambda tag_Cluster ids. After session smoke, at least one series should exist.
	Context("boundary observability recording rules", func() {

		It("should have zoa:access_availability after Access HTTP traffic", func() {
			query := `count(zoa:access_availability) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected zoa:access_availability from Access session-plane HTTP")
		})

		It("should have zoa:lambda_invocations_combined when Lambda traffic exists", func() {
			query := `count(zoa:lambda_invocations_combined) > 0`
			Eventually(func() bool {
				resp := thanosQuery(client, query)
				return resp.Status == "success" && len(resp.Data.Result) > 0
			}, "5m", "15s").Should(BeTrue(),
				"Expected combined Lambda invocation recording rule after api/worker/access traffic")
		})
	})

	// Recording rules only retain cluster (zoa.yaml). Discover clusters from
	// YACE Lambda errors (nilToZero), assert each rule on every RC/MC cluster.
	//
	// GC and reaper recording rule values are excluded — same ~5 min tick + YACE/CW
	// delay as in the infrastructure metrics comment. Definitions still verified in
	// recording_rules_test.go; GC/reaper alerts in alerts_test.go.
	Context("recording rule values", Ordered, func() {

		const (
			clusterTypeRegional   = "regional-cluster"
			clusterTypeManagement = "management-cluster"
		)

		var regionalClusters []string
		var managementClusters []string

		BeforeAll(func() {
			Eventually(func() bool {
				regionalClusters = discoverZOAClusters(client, clusterTypeRegional)
				managementClusters = discoverZOAClusters(client, clusterTypeManagement)
				return len(regionalClusters) > 0 && len(managementClusters) > 0
			}, "5m", "15s").Should(BeTrue(),
				"discover ZOA Lambda metric clusters for regional-cluster and management-cluster")
			GinkgoWriter.Printf("regional clusters: %v\n", regionalClusters)
			GinkgoWriter.Printf("management clusters: %v\n", managementClusters)
		})

		ruleNames := []string{
			"zoa:reconciler_last_run",
			"zoa:reconciler_tick_count",
			"zoa:lambda_invocations_combined",
			"zoa:lambda_errors_combined",
			"zoa:lambda_throttles_combined",
			"zoa:lambda_error_rate",
			"zoa:lambda_throttle_rate",
			"zoa:ta_success_rate",
			"zoa:ta_total_executions",
			"zoa:api_availability",
			"zoa:identity_bridge_hit_rate",
		}

		assertRuleOnAllClusters := func(rule string, clusters []string, role string) {
			Expect(clusters).NotTo(BeEmpty(), "expected at least one %s cluster", role)
			for _, cluster := range clusters {
				By(fmt.Sprintf("recording rule %s on %s cluster %q", rule, role, cluster))
				query := recordingRuleValuesQueryForCluster(rule, cluster)
				Eventually(func() bool {
					resp := thanosQuery(client, query)
					return resp.Status == "success" && len(resp.Data.Result) > 0
				}, "5m", "15s").Should(BeTrue(),
					"%s should produce values on %s cluster %q", rule, role, cluster)
			}
		}

		for _, rule := range ruleNames {
			rule := rule
			It("should have "+rule+" on every regional cluster", func() {
				assertRuleOnAllClusters(rule, regionalClusters, "regional")
			})
			It("should have "+rule+" on every management cluster", func() {
				assertRuleOnAllClusters(rule, managementClusters, "management")
			})
		}
	})
})
