[[ "${_HELPERS_SH:-""}" == "yes" ]] && return 0
_HELPERS_SH=yes

function print_global_variables() {
  debug "_JF_CWD=${_JF_CWD}"
  ####
  # JF_PATH_BASE is set when this program is run under Docker.
  debug "_JF_PATH_BASE=${_JF_PATH_BASE}"
  debug "_JF_PATH=${_JF_PATH}"
  debug "_JF_INFO=${_JF_INFO}"
  debug "_JF_DEBUG=${_JF_DEBUG}"
  debug "_JF_PERF=${_JF_PERF}"
}
