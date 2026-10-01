// Package targettype resolves ZOA target type (rc or mc) for TA registry filtering.
package targettype

import (
	"fmt"
	"os"
	"strings"
)

const (
	// EnvVar is the environment variable (matches TYPE in `zoa targets`).
	EnvVar = "ZOA_TARGET_TYPE"

	RC = "rc"
	MC = "mc"
)

// Resolve returns normalized target type: flag value, then EnvVar.
func Resolve(flagValue string) (string, error) {
	if v := normalize(flagValue); v != "" {
		return v, nil
	}
	if v := normalize(os.Getenv(EnvVar)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf(
		"target type is required (rc or mc): set --target-type or %s",
		EnvVar,
	)
}

// ResolveOptional returns normalized target type or empty when unset.
func ResolveOptional(flagValue string) string {
	v, err := Resolve(flagValue)
	if err != nil {
		return ""
	}
	return v
}

func normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case RC, MC:
		return s
	default:
		return ""
	}
}
