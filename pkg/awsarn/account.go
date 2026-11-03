package awsarn

import "strings"

// AccountID returns the 12-digit AWS account from a standard IAM or STS ARN.
// Example: arn:aws:sts::123456789012:assumed-role/role/session → 123456789012
func AccountID(arn string) string {
	if arn == "" {
		return ""
	}
	parts := strings.Split(arn, ":")
	if len(parts) < 5 {
		return ""
	}
	id := parts[4]
	if len(id) != 12 {
		return ""
	}
	for i := 0; i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return ""
		}
	}
	return id
}
