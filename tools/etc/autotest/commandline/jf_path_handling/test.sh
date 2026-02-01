#!/usr/bin/env bash
set -eu

# shellcheck disable=SC1090 source=lib/core.sh
. "${JF_BASEDIR}/lib/core.sh"

function run_testcase() {
  local _file_to_be_searched="${1}"
  local _path="${2}"
  local _expected_return_code="${3}"
  local _expected_output="${4:-""}"
  local _actual_output
  local _actual_err_file
  local _actual_return_code=0
  local _msg=""
  local _ret=1
  _actual_err_file="$(mktemp)"
  _path=$(eval "echo ${_path}")
  _expected_output=$(eval "echo ${_expected_output}")
  _actual_output=$(search_file_in "${_file_to_be_searched}" "${_path}" 2> "${_actual_err_file}") || {
    _actual_return_code=$?
  }
  if [[ ${_actual_return_code} -ne ${_expected_return_code} ]]; then
    _msg="Expected return code was ${_expected_return_code} but ${_actual_return_code} was returned"
  fi
  if [[ "${_actual_output}" != *"${_expected_output}"* ]]; then
    _msg="${_msg}\n'${_expected_output}' was expected to be found in output but not:\nActual output was:'${_actual_output}'"
  fi
  if [[ -z "${_msg}" ]]; then
    _ret=0
  else
    _msg="${_msg}\nstderr='$(cat "${_actual_err_file}")'\n"
  fi
  printf "${_msg}"
  return ${_ret}
}

_JF_PATH_BASE=${JF_PATH_BASE:-""}
export _JF_PATH_BASE

run_testcase "$@"
