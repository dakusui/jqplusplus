[[ "${_SHARED_SH:-""}" == "yes" ]] && return 0
_SHARED_SH=yes

# shellcheck disable=SC1090
# source = lib/logging.sh
source "${JF_BASEDIR}/lib/logging.sh"

function is_effectively_empty_array() {
  [[ "${#}" == 0 ]] && return 0
  [[ "${#}" == 1 && "${1}" == "" ]] && return 0
  return 1
}

function hashcode() {
  local _nodeentry="${1}"
  echo -n "${_nodeentry}" | md5sum | cut -f 1 -d ' '
}

function join_by() {
  local IFS="$1"
  shift
  echo "$*"
}

function trim() {
  local _data="${1}" _length="${2:-40}"
  if [[ "${#_data}" -gt "${_length}" ]]; then
    echo "${_data}"
    return 0
  fi
  echo "${_data:0:40}..."
}

function message() {
  local IFS=" "
  local _o
  _o="${1}"
  shift
  echo -e "${_o}" "$*" >&2
}

function abort() {
  print_stacktrace "ERROR:" "${@}"
  exit 1
}

# Use this to abort a function when you know that a failed procedure gives an stacktrace on a failure.
function abort_no_stacktrace() {
  message "ERROR:" "${@}"
  exit 1
}

function print_stacktrace() {
  local _message="${1}"
  shift
  message "${_message}" "${@}"
  local _i=0
  local _e
  while _e="$(caller $_i)"; do
    message "  at ${_e}"
    _i=$((_i + 1))
  done
}

function mktemp_with_content() {
  ####
  # This is a logic to check preceding procedure was successful.
  # shellcheck disable=SC2181
  [[ $? == 0 ]] || abort "Preceding command is suspected to be failed already."
  local _content="${1:?No content was given}"
  local _suffix="${2:?A suffix must be set}"
  local _dir
  _dir="${3:-$(_misctemp_files_dir)}"
  local _ret


  _ret="$(mktemp -p "${_dir}" --suffix="${_suffix}")"
  echo "${_content}" >"${_ret}"
  echo "${_ret}"
}

function is_localnode() {
  local _absfile="${1}"
  [[ "${_absfile}" == "${_JF_SESSION_DIR}/localnodes-"*/* ]] && return 0
  return 1
}

function mk_localnodedir() {
  local _absfile="${1}"
  local _ret
  _ret="${_JF_SESSION_DIR}/localnodes-$(hashcode "${_absfile}")"
  [[ -d "${_ret}" ]] || mkdir "${_ret}"
  echo "${_ret}"
}

function search_file_in() {
  local _target="${1}" _path="${2}" _path_base="${3:-${_JF_PATH_BASE}}"
  local _optional="no"
  local i
  if [[ "${_target}" == *\? ]]; then
    _optional="yes"
    _target="${_target%\?}"
  fi
  debug "begin: _target='${_target}', _path='${_path}', _path_base='${_path_base}', optional=${_optional}"
  IFS=':' read -r -a _arr <<<":${_path}"
  for i in "${_arr[@]}"; do
    local _ret
    _ret="$(echo "${i}/${_target}" | sed -E 's!/+!/!g')"
    if [[ -e "${_ret}" ]]; then
      debug "${_target} was found as '${_ret}' under '${i}'"
      echo "${_ret}"
      debug "end: _ret='${_ret}'"
      return 0
    fi
  done
  if [[ "${_optional}" == yes ]]; then
    mktemp_with_content '{}' '.json'
  else
    abort "File '${_target}' was not found in '${_path}'(cwd:'$(pwd)')"
  fi
}
