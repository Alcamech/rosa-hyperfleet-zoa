//go:build e2e

// CLI: zoa get, zoa output, zoa logs, zoa download — retrieve execution results.

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("zoa get", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			var execID string

			BeforeEach(func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				id, ok := exec["id"].(string)
				Expect(ok).To(BeTrue())
				Expect(id).NotTo(BeEmpty())
				execID = id
			})

			It("retrieves execution metadata in table format", func() {
				out, err := runZoa(tgt, "get", execID)
				Expect(err).NotTo(HaveOccurred(), out)
				Expect(out).To(ContainSubstring(execID[:8]))
				Expect(out).To(ContainSubstring("get_resource"))
				Expect(out).To(ContainSubstring("succeeded"))
			})

			It("retrieves execution in JSON format", func() {
				out, err := runZoa(tgt, "get", execID, "-o", "json")
				Expect(err).NotTo(HaveOccurred(), out)

				jsonStr := extractJSON(out)
				var exec struct {
					ID     string `json:"id"`
					Action string `json:"action"`
					Status string `json:"status"`
				}
				Expect(json.Unmarshal([]byte(jsonStr), &exec)).To(Succeed(), out)
				Expect(exec.ID).To(Equal(execID))
				Expect(exec.Action).To(Equal("get_resource"))
				Expect(exec.Status).To(Equal("succeeded"))

				var raw map[string]interface{}
				Expect(json.Unmarshal([]byte(jsonStr), &raw)).To(Succeed(), out)
				expectExecutionWithoutBoundarySessionID(raw)
			})

			It("retrieves execution with --include-output", func() {
				out, err := runZoa(tgt, "get", execID, "--include-output", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), out)

				jsonStr := extractJSON(out)
				var exec struct {
					ID     string      `json:"id"`
					Output interface{} `json:"output"`
				}
				Expect(json.Unmarshal([]byte(jsonStr), &exec)).To(Succeed(), out)
				Expect(exec.Output).NotTo(BeNil(), "output should be included")
			})

			It("retrieves execution with --include-logs", func() {
				out, err := runZoa(tgt, "get", execID, "--include-logs", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), out)

				jsonStr := extractJSON(out)
				var exec struct {
					ID   string `json:"id"`
					Logs string `json:"logs"`
				}
				Expect(json.Unmarshal([]byte(jsonStr), &exec)).To(Succeed(), out)
			})

			It("retrieves execution with --include-all", func() {
				out, err := runZoa(tgt, "get", execID, "--include-all", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), out)

				jsonStr := extractJSON(out)
				var exec struct {
					ID     string      `json:"id"`
					Output interface{} `json:"output"`
					Logs   string      `json:"logs"`
				}
				Expect(json.Unmarshal([]byte(jsonStr), &exec)).To(Succeed(), out)
			})

			It("--wait reconnects to an already-completed execution", func() {
				// execID is already succeeded (from BeforeEach); --wait should return immediately
				out, err := runZoa(tgt, "get", execID, "--wait", "--wait-timeout", "10s", "-o", "json")
				Expect(err).NotTo(HaveOccurred(), out)

				jsonStr := extractJSON(out)
				var exec struct {
					ID     string `json:"id"`
					Status string `json:"status"`
				}
				Expect(json.Unmarshal([]byte(jsonStr), &exec)).To(Succeed(), out)
				Expect(exec.ID).To(Equal(execID))
				Expect(exec.Status).To(Equal("succeeded"))
			})

			It("rejects an invalid execution ID", func() {
				out, err := runZoa(tgt, "get", "00000000-0000-0000-0000-000000000000")
				Expect(err).To(HaveOccurred())
				Expect(out).To(ContainSubstring("not found"))
			})
		})
	}
})

var _ = Describe("zoa output", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("retrieves raw TA output", func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				id := exec["id"].(string)

				out, err := runZoa(tgt, "output", id)
				Expect(err).NotTo(HaveOccurred(), out)
				Expect(out).NotTo(BeEmpty())
			})
		})
	}
})

var _ = Describe("zoa logs", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("retrieves execution logs", func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				id := exec["id"].(string)

				out, err := runZoa(tgt, "logs", id)
				Expect(err).NotTo(HaveOccurred(), out)
			})
		})
	}
})

var _ = Describe("zoa download", func() {
	for _, tgt := range targets {
		tgt := tgt

		Describe(tgt.Name, func() {
			It("downloads output to a file", func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				id := exec["id"].(string)

				tmpDir := GinkgoT().TempDir()
				outFile := filepath.Join(tmpDir, "output.json")

				// Note: -f for file path, not -o (which would be output format)
				out, err := runZoa(tgt, "download", id, "-f", outFile)
				Expect(err).NotTo(HaveOccurred(), out)

				data, err := os.ReadFile(outFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeEmpty())
			})

			It("downloads logs with --artifact logs", func() {
				exec := runAction(tgt, "get_resource", "--resource", "nodes")
				id := exec["id"].(string)

				tmpDir := GinkgoT().TempDir()
				outFile := filepath.Join(tmpDir, "logs.jsonl")

				out, err := runZoa(tgt, "download", id, "--artifact", "logs", "-f", outFile)
				Expect(err).NotTo(HaveOccurred(), out)

				data, err := os.ReadFile(outFile)
				Expect(err).NotTo(HaveOccurred())
				Expect(data).NotTo(BeEmpty())
			})
		})
	}
})
