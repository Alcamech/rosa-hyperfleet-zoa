//go:build e2e_monitoring

// Package e2e_monitoring validates ZOA observability: metrics in Thanos,
// recording rules loaded, alerting rules loaded, and no false-positive alerts.
//
// RHOBS_API_URL is required — missing it is a hard failure. Use test-e2e-zoa
// or test-e2e-zoa-smoke Makefile targets to run functional tests without monitoring.
//
// Run standalone:
//
//	RHOBS_API_URL=https://xxx.execute-api.us-east-1.amazonaws.com/prod \
//	  AWS_PROFILE=rrp-rc go test -tags e2e_monitoring ./test/e2e-monitoring/... -v
package e2e_monitoring

import (
	"net/http"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var (
	rhobsAPIURL string
	awsRegion   string
	awsProfile  string
	client      *rhobsClient
)

func TestE2EMonitoring(t *testing.T) {
	if os.Getenv("RHOBS_API_URL") == "" {
		t.Fatal("RHOBS_API_URL not set — monitoring tests require it; use test-e2e-zoa to skip monitoring")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "ZOA Monitoring E2E Suite")
}

var _ = BeforeSuite(func() {
	rhobsAPIURL = os.Getenv("RHOBS_API_URL")
	awsRegion = envOrDefault("AWS_REGION", "us-east-1")
	// Reuse ZOA_RC_AWS_PROFILE — the RHOBS API Gateway restricts query
	// endpoints to the RC account, which is the same account the ZOA E2E
	// suite uses for RC targets. Falls back to rrp-rc (same default as
	// test/e2e/helpers_test.go discoverTargets).
	awsProfile = envOrDefault("ZOA_RC_AWS_PROFILE", "rrp-rc")

	client = newRHOBSClient(rhobsAPIURL, awsRegion, awsProfile)

	By("Verifying Thanos query endpoint is reachable via RHOBS API Gateway")
	Eventually(func() int {
		resp, err := client.get("/api/v1/query?query=up")
		if err != nil {
			GinkgoWriter.Printf("Thanos probe error: %v\n", err)
			return 0
		}
		return resp.StatusCode
	}, "1m", "5s").Should(Equal(http.StatusOK),
		"Thanos /api/v1/query should be reachable through RHOBS API Gateway")

	By("Verifying Thanos rules endpoint is reachable")
	Eventually(func() int {
		resp, err := client.get("/api/v1/rules")
		if err != nil {
			GinkgoWriter.Printf("Thanos rules probe error: %v\n", err)
			return 0
		}
		return resp.StatusCode
	}, "1m", "5s").Should(Equal(http.StatusOK),
		"Thanos /api/v1/rules should be reachable through RHOBS API Gateway")
})
