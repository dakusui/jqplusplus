[[ "${_LOGGING_SH:-""}" == "yes" ]] && return 0
_LOGGING_SH=yes

function is_debug_enabled() {
  [[ ${_JF_DEBUG:-""} == "enabled" ]] && return 0
  return 1
}

function debug() {
  if is_debug_enabled; then
    message "DEBUG: $(date '+%Y-%m-%d %H:%M:%S.%3N'): ${FUNCNAME[1]}:" "${@}"
  fi
}

function is_info_enabled() {
  [[ ${_JF_INFO:-""} == "enabled" ]] && return 0
  return 1
}

function info() {
  if is_debug_enabled; then
    message "INFO: $(date '+%Y-%m-%d %H:%M:%S.%3N'): ${FUNCNAME[1]}:" "${@}"
  elif is_info_enabled; then
    message "INFO: unknown: ${FUNCNAME[1]}:" "${@}"
  fi
}


function is_perf_enabled() {
  [[ ${_JF_PERF:-""} == "enabled" ]] && return 0
  return 1
}

function perf() {
  if is_perf_enabled; then
    message "PERF: $(date '+%Y-%m-%d %H:%M:%S.%3N'): ${FUNCNAME[1]}:" "${@}"
  fi
}

function error() {
    message "ERROR: $(date '+%Y-%m-%d %H:%M:%S.%3N'): ${FUNCNAME[1]}:" "${@}"
}
