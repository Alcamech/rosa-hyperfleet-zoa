package cli

import (
	"fmt"
	"regexp"
)

var (
	sessionDispatchedLineRE = regexp.MustCompile(`(?m)^✓ ([^\s]+) \[`)
	sessionJoinHintRE       = regexp.MustCompile(`zoa session join ([^\s]+)`)
)

// ParseCompoundSessionIDFromCLIOutput extracts the compound session id
// (deployment/raw-id) from combined stdout/stderr after session start or join with --exec-script
// or after interactive start (progress lines on stderr).
//
// Intended for automation that runs session start --exec-script without -o json: stdout is the
// Exec transcript; session metadata appears on stderr (✓ line and post-Exec hints).
func ParseCompoundSessionIDFromCLIOutput(output string) (string, error) {
	if m := sessionDispatchedLineRE.FindStringSubmatch(output); len(m) == 2 {
		if _, _, err := ParseSessionID(m[1]); err == nil {
			return m[1], nil
		}
	}
	matches := sessionJoinHintRE.FindAllStringSubmatch(output, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if len(matches[i]) != 2 {
			continue
		}
		if _, _, err := ParseSessionID(matches[i][1]); err == nil {
			return matches[i][1], nil
		}
	}
	return "", fmt.Errorf("compound session id not found in CLI output")
}
