[[ "${_DEPENDENCY_SH:-""}" == "yes" ]] && return 0
_DEPENDENCY_SH=yes

function _encode_filename() {
  local _filename="${1}" _dependency_space="${2}"
  local _ret
  _ret="$(_inprogress_files_dir)/${_dependency_space}-$(hashcode "${_filename}")"
  echo "${_ret}"
}

function _mark_as_in_progress() {
  local _filename="${1}" _dependency_space="${2}"
  local _encoded_filename
  _encoded_filename="$(_encode_filename "${_filename}" "${_dependency_space}")"
  touch "${_encoded_filename}"
  debug "'${_filename}' was marked as in progress with: '${_encoded_filename}'"
  [[ -e "${_encoded_filename}" ]] || abort "Failed to create a mark file!"
}

function _unmark_as_in_progress() {
  local _filename="${1}" _dependency_space="${2}"
  local _encoded_filename
  debug "check: unmark: '${_filename}'"
  _encoded_filename="$(_encode_filename "${_filename}" "${_dependency_space}")"
  rm "${_encoded_filename}" || {
    message "INFO: A markfile:'${_encoded_filename}' for ${_dependency_space}:${_filename} was not found."
  }
}

function _is_in_progress() {
  local _filename="${1}" _dependency_space="${2}"
  local _encoded_filename
  _encoded_filename="$(_encode_filename "${_filename}" "${_dependency_space}")"
  if [[ -e "${_encoded_filename}" ]]; then
    debug "'${_filename}'(${_encoded_filename}) is in progress"
    return 0
  fi
  debug "'${_filename}'(${_encoded_filename}) is NOT in progress"
  return 1
}

function _check_cyclic_dependency() {
  local _in="${1}" _dependency_space="${2}"
  if _is_in_progress "${_in}" "${_dependency_space}"; then
    abort "Cyclic ${_dependency_space} was detected in:'${_in}'"
  else
    debug "mark '${_in}' as in progress"
    _mark_as_in_progress "${_in}" "${_dependency_space}"
  fi
}
