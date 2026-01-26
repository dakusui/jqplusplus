#!/bin/bash -eu
set -E -eu -o pipefail
shopt -s inherit_errexit

_JF="${1:?Specify 'jf' to test in absolute path}"
_TEST_ROOT_DIR="${2:?Specify a directory in which test dirs are stored.}"
_TARGET_TESTS="${3:-*}"
_JF_PATH=.:${_TEST_ROOT_DIR}/base
JF_BASEDIR="$(dirname "$(dirname "${0}")")"
export JF_BASEDIR

# shellcheck disable=SC1090 source=lib/core.sh
source "${JF_BASEDIR}/lib/core.sh"

function run_normal_test() {
  local _testfile="${1}"
  local _dirname
  local _diff
  local _ret=1
  _dirname="$(dirname ${_testfile})"
  ${_JF} "${_dirname}/input.json" >"${_dirname}/test-output".json 2>"${_dirname}/test-error.txt" || return 1
  {
    diff <(jq -S . "${_dirname}/expected.json") <(jq -S . "${_dirname}/test-output.json") >"${_dirname}/test-output.diff"
  }
  _diff=$(cat "${_dirname}/test-output.diff") || return 1
  if [[ -z ${_diff} ]]; then
    message "PASSED"
    _ret=0
  else
    _ret=1
    message "FAILED: diff--->
    ${_diff}
    ---"
  fi
  return ${_ret}
}

function run_commandline_test() {
  local _testfile="${1}"
  local _dirname _diff _option
  local _ret=1
  _dirname="$(dirname "${_testfile}")"
  _option="$(jq -r -c '.option' "${_testfile}")"
  ${_JF} "${_option}" >"${_dirname}/test-output".json 2>"${_dirname}/test-error.txt" || return 1
  local _mismatches="${_dirname}/test-output.diff"
  {
    local _expected
    local _error
    local i
    _error="$(cat "${_dirname}/test-error.txt")"
    IFS=$'\n' read -d '' -a _expected <"${_dirname}/expected.txt"
    for i in "${_expected[@]}"; do
      if [[ ${_error} == *"${i}"* ]]; then
        continue
      else
        echo "${i}" >>"${_mismatches}"
      fi
    done
  }
  if [[ -f ${_mismatches} ]]; then
    local _diff
    _diff=$(cat "${_mismatches}") || return
    _ret=1
    message "FAILED: missing messages in stderr--->
    ${_diff}
    ---"
  else
    message "PASSED"
    _ret=0
  fi
  return ${_ret}
}

function run_negative_test() {
  local _testfile="${1}"
  local _dirname
  local _ret=1
  _dirname="$(dirname ${_testfile})"
  ${_JF} "${_dirname}/input.json" >"${_dirname}/test-output".json 2>"${_dirname}/test-error.txt" && {
    message "ABNORMAL EXIT WAS EXPECTED BUT IT WAS NORMAL:<${_JF}>:<$?>"
    return 1
  }
  local _mismatches="${_dirname}/test-output.diff"
  {
    local _expected _error i
    _error="$(cat "${_dirname}/test-error.txt")"
    IFS=$'\n' read -d '' -a _expected <"${_dirname}/expected.txt"
    for i in "${_expected[@]}"; do
      if [[ ${_error} =~ ${i} ]]; then
        continue
      else
        echo "< ${i}" >>"${_mismatches}"
      fi
    done
  }
  [[ -e "${_dirname}/unexpected.txt" ]] && {
    local _expected _error i
    _error="$(cat "${_dirname}/test-error.txt")"
    IFS=$'\n' read -d '' -a _expected <"${_dirname}/unexpected.txt"
    for i in "${_expected[@]}"; do
      if [[ ${_error} =~ ${i} ]]; then
        echo "> ${i}" >>"${_mismatches}"
      else
        continue
      fi
    done
  }
  if [[ -f ${_mismatches} ]]; then
    local _diff
    _diff=$(cat "${_mismatches}") || return
    _ret=1
    message "FAILED: missing messages in stderr--->
    ${_diff}
    ---"
  else
    message "PASSED"
    _ret=0
  fi
  return ${_ret}
}

run_ignore_test() {
  message "IGNORED"
}

function _failure_message() {
  local _expected_validity_level="${1}"
  local _schema="${2}"
  local _expectation="${3}"
  message "(Expectation:${_expected_validity_level}):Expected to be ${_expectation} with ${_schema} schema but not:'${_dirname/input.txt/}'"
  cat "${_dirname}/test-output-${_schema}".txt >&2
}

function run_validation_test() {
  local _testfile="${1}"
  local _actual_strict
  local _actual_lenient
  local _expected_validity_level
  local _ret=0
  _expected_validity_level=$(jq -r -c ".expectation.validityLevel" "${_testfile}")
  _dirname="$(dirname ${_testfile})"
  ${_JF} "${_dirname}/input.json" --nested-templating-levels=0 --validation=lenient &>"${_dirname}/test-output-lenient".txt
  _actual_lenient=$?
  ${_JF} "${_dirname}/input.json" --nested-templating-levels=0 --validation=strict &>"${_dirname}/test-output-strict".txt
  _actual_strict=$?
  if [[ "${_expected_validity_level}" == "strict" ]]; then
    if [[ "${_actual_lenient}" != 0 ]]; then
      _failure_message "${_expected_validity_level}" "lenient" "valid"
      _ret=1
    fi
    if [[ "${_actual_strict}" != 0 ]]; then
      _failure_message "${_expected_validity_level}" "strict" "valid"
      _ret=1
    fi
  elif [[ "${_expected_validity_level}" == "lenient" ]]; then
    if [[ "${_actual_lenient}" != 0 ]]; then
      _failure_message "${_expected_validity_level}" "lenient" "valid"
      _ret=1
    fi
    if [[ "${_actual_strict}" == 0 ]]; then
      _failure_message "${_expected_validity_level}" "strict" "INVALID"
      _ret=1
    fi
  elif [[ "${_expected_validity_level}" == "invalid" ]]; then
    if [[ "${_actual_lenient}" == 0 ]]; then
      _failure_message "${_expected_validity_level}" "lenient" "INVALID"
      _ret=1
    fi
    if [[ "${_actual_strict}" == 0 ]]; then
      _failure_message "${_expected_validity_level}" "strict" "INVALID"
      _ret=1
    fi
  else
    message "Unknown expectation validity level:'${_expected_validity_level}' was specified."
    return 1
  fi
  if [[ ${_ret} == 0 ]]; then
    message "PASSED"
  else
    message "FAILED"
  fi
  return ${_ret}
}

function _run_testcase() {
  local _executor="${1}"
  shift
  bash "${_executor}" "${@}"
  return $?
}

function run_suite_test() {
  local _testfile="${1}"
  local _testfile_dir="${_testfile}"
  local _executor
  local _numtestcases
  local _numfailed=0
  local _ret=0
  _testfile_dir="$(dirname ${_testfile}"")"
  _executor="${_testfile_dir}/$(jq -r -c '.args.executor' "${_testfile}")"
  _numtestcases="$(jq -r -c '.args.testCases | length' ${_testfile})"
  message ""
  for i in $(seq 0 $((_numtestcases - 1))); do
    message -n "  Running test case [${i}]"
    # shellcheck disable=SC2046
    _run_testcase "${_executor}" $(jq -r -c ".args.testCases[${i}].args[]" "${_testfile}") || {
      _ret=1
      _numfailed=$((_numfailed + 1))
      message "...FAILED"
      continue
    }
    message "...PASSED"
  done
  message "  ${_numfailed} test cases (out of ${_numtestcases}) have failed"
  if [[ ${_ret} == 0 ]]; then
    message "PASSED"
  else
    message "FAILED"
  fi
  return ${_ret}
}

function runtest() {
  local _test_json="${1}"
  local _dirname="${_test_json%/test.json}"
  local _test_type
  _test_type="$(jq -r -c ".testType" "${_test_json}")" || return 1
  message -n "executing test(${_test_type}):'${_dirname#${_TEST_ROOT_DIR}/}':..."

  export JF_PATH
  JF_PATH="${_JF_PATH}"
  "run_${_test_type}_test" "${_test_json}" || return 1
}

function clean() {
  find "${_TEST_ROOT_DIR}" -name 'test-output*' -exec rm {} \;
}

function main() {
  local -a failed=()
  local passed=0
  local skipped=0
  local run=0
  local numtests=0
  local -a tests
  local _target_tests="${_TARGET_TESTS}"
  message "${_TEST_ROOT_DIR}:target=${_TARGET_TESTS}"
  _target_tests="*${_TARGET_TESTS}*"
  while IFS='' read -r -d '' i; do
    numtests=$((numtests + 1))
    tests+=("${i}")
  done < <(find "${_TEST_ROOT_DIR}" -type f -name test.json -print0)
  for i in "${tests[@]}"; do
    if [[ "x${i}" == x${_target_tests} ]]; then
      run=$((run + 1))
      {
        runtest "${i}" && passed=$((passed + 1))
      } || failed+=("${i}")
    else
      skipped=$((skipped + 1))
      message "Skipping ${i}"
    fi
  done

  if [[ ${#failed[@]} == 0 ]]; then
    message "No test failed (total=${numtests}; passed=${passed}; skipped=${skipped})"
  else
    message "${#failed[@]} test(s) FAILED (total=${numtests}; passed=${passed}; run=${run}; skipped=${skipped})"
    for i in "${failed[@]}"; do
      message "${i}"
    done
    return 1
  fi
}

clean || exit 1
main || exit 1
