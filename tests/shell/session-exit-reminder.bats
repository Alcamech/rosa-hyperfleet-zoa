#!/usr/bin/env bats

load test-env

setup() {
  REPO_ROOT="$(zoa_shell_test_repo_root)"
  BANNER="$(zoa_shell_test_banner_path)"
  zoa_shell_test_export_banner
  SESSION_ID_RC="${REPO_ROOT}/boundary/home-sre/.bashrc.d/09-zoa-session-id.bashrc"
  REMINDER="${REPO_ROOT}/boundary/home-sre/.bashrc.d/99-session-exit-reminder.bashrc"
  PROMPT="${REPO_ROOT}/boundary/home-sre/.bashrc.d/10-zoa-prompt.bashrc"
}

@test "compound session id combines deployment and raw uuid" {
  run env \
    ZOA_DEPLOYMENT="us-east-1-eph-046f5f15" \
    ZOA_SESSION_ID="0414dd71-ccc7-40bc-8c5b-072cd5982e96" \
    bash -c '
      source "$1"
      zoa_compound_session_id
    ' bash "${SESSION_ID_RC}"

  [ "${status}" -eq 0 ]
  [ "${output}" = "us-east-1-eph-046f5f15/0414dd71-ccc7-40bc-8c5b-072cd5982e96" ]
}

@test "prompt primary line uses full compound session id" {
  run env \
    ZOA_DEPLOYMENT="us-east-1-eph-046f5f15" \
    ZOA_SESSION_ID="0414dd71-ccc7-40bc-8c5b-072cd5982e96" \
    bash -c '
      source "$1"
      source "$2"
      zoa_prompt_primary
    ' bash "${SESSION_ID_RC}" "${PROMPT}"

  [ "${status}" -eq 0 ]
  [ "${output}" = "sessionId:us-east-1-eph-046f5f15/0414dd71-ccc7-40bc-8c5b-072cd5982e96" ]
}

@test "session terminate hint includes deployment and session id" {
  run env \
    ZOA_BOUNDARY_BANNER="${BANNER}" \
    ZOA_BOUNDARY_BANNER_PATH="${BANNER}" \
    ZOA_DEPLOYMENT="us-east-1-eph" \
    ZOA_SESSION_ID="sess-abc123" \
    ZOA_SESSION_DEADLINE="2030-01-01T12:00:00Z" \
    ZOA_SESSION_IDLE_TIMEOUT_SECONDS="3600" \
    bash -c '
      source "$1"
      source "$2"
      zoa_boundary_print_session_terminate_hint
    ' bash "${SESSION_ID_RC}" "${REMINDER}"

  [ "${status}" -eq 0 ]
  [[ "${output}" == *"zoa session terminate us-east-1-eph/sess-abc123"* ]]
  [[ "${output}" == *"zoa session join us-east-1-eph/sess-abc123"* ]]
  [[ "${output}" == *"zoa session list us-east-1-eph"* ]]
  [[ "${output}" == *"All operators (audit):"* ]]
  [[ "${output}" == *"zoa session history us-east-1-eph"* ]]
  [[ "${output}" == *"Hard termination:"* ]]
  [[ "${output}" == *"Inactivity termination:"* ]]
  [[ "${output}" == *"still running"* ]]
  [[ "${output}" != *"-d "* ]]
}

@test "session terminate hint is silent when ZOA_SESSION_ID unset" {
  run env \
    ZOA_DEPLOYMENT="us-east-1-eph" \
    bash -c '
      source "$1"
      source "$2"
      zoa_boundary_print_session_terminate_hint
    ' bash "${SESSION_ID_RC}" "${REMINDER}"

  [ "${status}" -eq 0 ]
  [ "${output}" = "" ]
}

@test "non-interactive shell does not install EXIT trap" {
  run env \
    ZOA_SESSION_ID="sess-abc123" \
    bash -c '
      source "$1"
      trap -p EXIT
    ' bash "${REMINDER}"

  [ "${status}" -eq 0 ]
  [[ "${output}" != *"zoa_boundary_print_session_terminate_hint"* ]]
}

@test "EXIT trap prints session terminate hint on shell exit" {
  run env \
    ZOA_BOUNDARY_BANNER="${BANNER}" \
    ZOA_BOUNDARY_BANNER_PATH="${BANNER}" \
    ZOA_DEPLOYMENT="us-east-1-eph" \
    ZOA_SESSION_ID="sess-abc123" \
    bash -c '
      source "$1"
      source "$2"
      trap zoa_boundary_print_session_terminate_hint EXIT
      exit
    ' bash "${SESSION_ID_RC}" "${REMINDER}"

  [ "${status}" -eq 0 ]
  [[ "${output}" == *"zoa session terminate us-east-1-eph/sess-abc123"* ]]
}
