[[ "${_NODEPOOL_SH:-""}" == "yes" ]] && return 0
_NODEPOOL_SH=yes

function nodepool_prepare() {
  local _pooldir
  debug "begin"
  _pooldir="${_JF_SESSION_DIR}/nodepool"
  mkdir -p "${_pooldir}"
  echo "${_pooldir}"
  debug "end"
}

function define_nodeentry_reader() {
  local _driver_funcname="${1}" _pooldir="${2}"
  readonly _NODEPOOL_SH_DRIVER_FUNCNAME="${_driver_funcname}"
  export _NODEPOOL_SH_DRIVER_FUNCNAME
  function read_nodeentry() {
    local _nodeentry="${1}" _validation_mode="${2}" _path="${3}"
    "${_NODEPOOL_SH_DRIVER_FUNCNAME}" "${_nodeentry}" "${_validation_mode}" "${_path}"
    [[ $? == 0 ]] || return 1
  }
  debug "read_nodeentry was defined:$(type read_nodeentry)"
}

function nodepool_read_nodeentry() {
  local _nodeentry="${1}" _validation_mode="${2}" _path="${3}" _pooldir="${4:-${_JF_POOL_DIR}}"
  local _cache
  perf "begin: '${_nodeentry}'"
  _nodeentry="$(_normalize_nodeentry "${_nodeentry}" "${_path}")"
  [[ $? == 0 ]] || {
    error "NODEENTRY_NOT_FOUND: '${_nodeentry}' was not found in '${_path}'"
    return 1
  }
  _cache="${_pooldir}/$(hashcode "${_nodeentry}")"
  _check_cyclic_dependency "${_nodeentry}" inheritance
  echo "- <${_nodeentry}>" >>"$(_misctemp_files_dir_nodepool_logfile)"
  if [[ -e "${_cache}" ]]; then
    perf "Cache hit for node entry: '${_nodeentry}'"
  else
    perf "Cache miss for node entry: '${_nodeentry}'"
    read_nodeentry "${_nodeentry}" "${_validation_mode}" "${_path}" >"${_cache}"
    [[ $? == 0 ]] || return 1
  fi
  cat "${_cache}"
  _unmark_as_in_progress "${_nodeentry}" inheritance
  perf "end: '${_nodeentry}' (cached by:'${_cache}')"
}

function _normalize_nodeentry() {
  local _nodeentry="${1}"
  local _path="${2}"
  local _specifier _absfile
  mapfile -t -d ';' _specifier <<<"${_nodeentry};;"
  _absfile="$(search_file_in "${_specifier[0]}" "${_path}")"
  _absfile="$(echo "${_absfile}" | sed -E 's!^(./)+!!g')"
  echo "${_absfile};${_specifier[1]};$(join_by ';' "${_specifier[@]:2}")" | sed -E 's/\;*$/;;/g'
}

function jsonize() {
  local _absfile="${1}" _processor="${2}" _args="${3:-""}"
  local _ret
  function __jsonize_read_file_if_possible() {
    local _f="${1}"
    [[ -f "${_f}" ]] && {
      cat "${_f}" || echo "(failed to read: ${_f})"
      return 0
    }
    echo "(not found: ${_f})"
  }
  _ret="$(_jsonize "${_absfile}" "${_processor}" "${_args}" 2>/dev/null)" ||
    abort "Malformed JSON was given:\n  path: '${_absfile}'\n  processor: '${_processor}'\n  args: '${_args}'\n  content: '$(__jsonize_read_file_if_possible "${_absfile}")'"
  echo "${_ret}"
}

function _jsonize() {
  local _absfile="${1}" _processor="${2}" _args="${3:-""}"
  local _ret
  debug "in: '${_absfile}' '${_processor}' '${_args}'"
  local -a _args_array
  mapfile -t _args_array <<<"$(IFS=';' && echo "${_args}" | tr ';' $'\n' | grep -v -E '^$')"
  is_effectively_empty_array "${_args_array[@]}" && _args_array=()
  if [[ ${_processor} == "" ]]; then
    _processor="$(resolve_processor "${_absfile}")"
  fi
  if [[ "${_processor}" == SOURCE ]]; then
    # Only number of files matters and it is safe to use ls here.
    # shellcheck disable=SC2012
    cp "${_absfile}" "$(_sourced_files_dir)/$(ls "$(_sourced_files_dir)" | wc -l)"
    _ret="{}"
  else
    debug "_processor='${_processor}', _absfile='${_absfile}',_args=${_args}"
    export _path
    _ret="$(${_processor} "${_absfile}" "${_args_array[@]}" | jq .)" ||
      abort "Failed to parse '${_absfile}' with '${_processor}'(args:${_args_array[*]}):(2)"
    unset _path
  fi
  is_debug_enabled && debug "output: ${_ret}"
  echo "${_ret}"
}
