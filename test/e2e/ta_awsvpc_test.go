//go:build e2e

// TA: list_vpc_endpoints, describe_vpc_endpoint (aws-api scope)

package e2e

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa run VPC endpoint aws-api actions", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			BeforeEach(func() {
				skipUnlessLiveActions(tgt, "list_vpc_endpoints", "describe_vpc_endpoint")
			})

			It("lists VPC endpoints and describes the first one, if any exist", func() {
				list := runAction(tgt, "list_vpc_endpoints")
				out := outputMap(list)
				Expect(out).To(HaveKey("endpoints"))
				Expect(out).To(HaveKey("count"))

				endpoints, _ := out["endpoints"].([]interface{})
				if len(endpoints) == 0 {
					// RC/MC are documented as fully-private (VPC-only) clusters, so
					// interface endpoints are expected in practice, but don't hard-fail
					// the whole suite on an account/region that genuinely has none.
					Skip("no VPC endpoints found in this account/region — nothing to describe")
				}

				ep, ok := endpoints[0].(map[string]interface{})
				Expect(ok).To(BeTrue())
				id, ok := ep["id"].(string)
				Expect(ok).To(BeTrue())
				Expect(id).To(HavePrefix("vpce-"))

				describe := runAction(tgt, "describe_vpc_endpoint", "--name", id)
				detail := outputMap(describe)
				Expect(detail["id"]).To(Equal(id))
			})

			It("rejects a malformed VPC endpoint ID", func() {
				out := runActionExpectFailure(tgt, "describe_vpc_endpoint", "--name", "not-a-vpce-id")
				Expect(out).To(ContainSubstring("must start with"))
			})

			It("requires a name", func() {
				out := runActionExpectFailure(tgt, "describe_vpc_endpoint")
				Expect(out).To(ContainSubstring("name"))
			})
		})
	}
})
