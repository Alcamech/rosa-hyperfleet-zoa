# Shared env for boundary shell tests (sourced from bats setup).
# Production uses /etc/zoa-boundary/banner.sh; tests point at the repo copy.

zoa_shell_test_repo_root() {
  cd "${BATS_TEST_DIRNAME}/../.." && pwd
}

zoa_shell_test_banner_path() {
  echo "$(zoa_shell_test_repo_root)/boundary/zoa-boundary-banner.sh"
}

zoa_shell_test_export_banner() {
  local banner
  banner="$(zoa_shell_test_banner_path)"
  export ZOA_BOUNDARY_BANNER_PATH="${banner}"
  export ZOA_BOUNDARY_BANNER="${banner}"
}

# util-linux `script` is required in the boundary image but may be absent on dev laptops.
zoa_shell_test_stub_bin_dir() {
  local stub_dir="${BATS_TMPDIR}/zoa-shell-stub-bin"
  mkdir -p "${stub_dir}"
  if [[ ! -x "${stub_dir}/script" ]]; then
    printf '%s\n' '#!/bin/sh' 'exit 0' > "${stub_dir}/script"
    chmod +x "${stub_dir}/script"
  fi
  echo "${stub_dir}"
}
