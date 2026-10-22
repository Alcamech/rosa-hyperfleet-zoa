// Temporary cross-repo compat while rosa-seaplane-zoa boundary lands before rosa-seaplane TF renames
// deployment_target → target_type (ZOA_DEPLOYMENT_TARGET → ZOA_TARGET_TYPE).
//
// Search: ROSAENG-68805, targettype_compat, coalesceTargetTypeEnv, ZOA_DEPLOYMENT_TARGET
package config

import "os"

// legacyTargetTypeEnv is set by pre-boundary rosa-seaplane Terraform (var.deployment_target).
const legacyTargetTypeEnv = "ZOA_DEPLOYMENT_TARGET"

// coalesceTargetTypeEnv prefers ZOA_TARGET_TYPE; falls back to legacy ZOA_DEPLOYMENT_TARGET.
//
// TODO(ROSAENG-68805): remove legacy env fallback after seaplane main provisions ZOA_TARGET_TYPE on all Lambdas.
func coalesceTargetTypeEnv() string {
	if v := normalizeTargetType(os.Getenv(targetTypeEnv)); v != "" {
		return v
	}
	return normalizeTargetType(os.Getenv(legacyTargetTypeEnv))
}
