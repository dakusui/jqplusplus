[[ "${_TEMPLATING_SH:-""}" == "yes" ]] && return 0
_TEMPLATING_SH=yes

function perform_templating() {
  local _src_file="${1}" _levels="${2}"
  local _content _ret
  perf "begin"
  # define builtin functions such as "ref", "self"
  _define_builtin_functions
  # source files, for which SOURCE directive is specified
  _source_files
  _perform_templating "$(cat "${_src_file}")" "${_levels}"
  perf "end"
}

function _perform_templating() {
  local _content="${1}" _levels="${2}"
  _content="$(_perform_templating_key_side "${_content}")"
  _content="$(_perform_templating_value_side "${_content}" "${_levels}")"
  echo "${_content}"
}

function _perform_templating_key_side() {
  local _content="${1}"
  local _ret="${_content}"
  local -a _entries
  # Shorter path comes earlier than longer.
  debug "begin"
  mapfile -t _entries < <(_keys_to_perform_templating "${_ret}")
  for _each in "${_entries[@]}"; do
    local _p _key _path _templated_key _entry
    perf "processing: entry:'${_each}'"
    IFS=$'\t' read -r -a _entry <<<"${_each}"
    _p="${_entry[2]:-""}"
    _key="${_entry[1]}"
    _path="${_entry[0]}"
    _templated_key="$(_render_text_node "${_key}" "${_p}" "${_ret}")"
    _ret="$(jq -r -c -M -n "input|${_p}.${_templated_key}=input" <(echo "${_ret}") <(echo "${_ret}" | jq -r -c -M "${_path}"))"
    _ret="$(echo "${_ret}" | jq -r -c -M 'del('"${_path}"')')"
  done
  echo "${_ret}"
  debug "end"
}

# $ echo '{"k1":"v","k2":[{"k21":"w"},{"k22":"x"}]}'
#   | jq -c -M '
#   . |paths
#     |. as $p
#     |$p    |.[0:-1]  |. as $path
#     |$p[-1]|. as $key|select(type=="string" and (startswith("eval:") or
#                                                  startswith("template:")))
#     |[$p,$key,$path]'
#   | jq -r -c -s -L lib '#---
#   import "shared" as shared;
#   . |sort_by(.0)
#     |sort_by(.0|length)
#     |reverse
#     |.[]
#     |[shared::path2pexp(.[0]),.[1],shared::path2pexp(.[2])]
#     |@tsv'
# [".\"k2\"[1].\"k22\"", "k22",       ".\"k2\"[1]"]
# [".\"k2\"[1]",         1,           ".\"k2\""]
# [".\"k2\"[0].\"k21\"", "k21",       ".\"k2\"[0]"]
# [".\"k2\"[0]",         0,           ".\"k2\""]
# [".\"k2\"",            "k2",        ""]
# [".\"k1\"",            "k1",        ""]
#  ^                     ^            ^
#  |                     |            |
#  |                     |            +--- [2]: P:    The path to the parent of the key [1]
#  |                     +---------------- [1]: KEY:  The key string on which text rendering should happen.
#  +-------------------------------------- [0]: PATH: The path to the key [1] itself.
#                                                     Remove this path from the current object.
#  1. P[KEY] = getpath(PATH)
#  2. del(PATH)
function _keys_to_perform_templating() {
  local _content="${1}"
  echo "${_content}" |
  jq -c -M '
     . |paths
       |. as $p
       |$p    |.[0:-1]  |. as $path
       |$p[-1]|. as $key|select(type=="string" and (startswith("eval:") or
                                                    startswith("template:")))
       |[$p,$key,$path]'   |
  jq -r -c -s -L "${JF_BASEDIR}/lib" '#---
     import "shared" as shared;
     . |sort_by(.0)
       |sort_by(.0|length)
       |reverse
       |.[]
       |[shared::path2pexp(.[0]),.[1],shared::path2pexp(.[2])]
       |@tsv'
}

function _perform_templating_value_side() {
  local _content="${1}" _levels="${2}"
  local _ret="${_content}" _c="${_levels}"
  local -a _entries
  # Shorter path comes earlier than longer.
  debug "begin"
  while [[ "${_c}" -ge 0 || "${_levels}" == -1 ]]; do
    local _each
    mapfile -t _entries < <(_string_node_entries_to_perform_templating "${_ret}")
    # shellcheck disable=SC2015
    is_effectively_empty_array "${_entries[@]}" && break || :
    if [[ "${_c}" -eq 0 ]]; then
      error "Templating has been repeated ${_levels} time(s) but it did not finish.: Entries left untemplated are: [${_entries[*]}]"
    fi
    perf "begin loop: remaining: ${_c}"
    for _each in "${_entries[@]}"; do
      local _node_path _node_value _templated_node_value _entry
      perf "processing: entry:'${_each}'"
      IFS=$'\t' read -r -a _entry <<<"${_each}"
      _node_path="${_entry[0]}"
      _node_value="${_entry[1]}"
      debug "processing: nodepath:'${_node_path}', nodevalue:'${_node_value}'"
      _templated_node_value="$(_render_text_node "${_node_value}" "${_node_path}" "${_ret}")"
      _ret="$(jq -r -c -n "input|${_node_path}=input" <(echo "${_ret}") <(echo "${_templated_node_value}"))"
    done
    perf "end loop: remaining: ${_c}"
    _c=$((_c - 1))
  done
  echo "${_ret}"
  debug "end"
}

function _string_node_entries_to_perform_templating() {
  local _content="${1}"
  echo "${_content}" | jq -r -c -L "${JF_BASEDIR}/lib" '#---
import "shared" as shared;
. as $content
|[paths(shared::scalars_or_empty
       |select(type=="string" and (startswith("eval:") or
                                   startswith("template:"))))]
       |sort
       |sort_by(length)
       |.[]
       |. as $p
       |$content
       |[shared::path2pexp($p),getpath($p)]
       |@tsv'
}

function _render_text_node() {
  local _node_value="${1}"
  local _path="${2}"         # DO NOT REMOVE: This local variable is referenced by built-in functions invoked on 'templating' stage.
  local _self_content="${3}" # DO NOT REMOVE: This local variable is referenced by built-in functions invoked on 'templating' stage.
  local _mode="raw" _quote="yes" _ret_code=0 _expected_type="string" _body _ret
  if [[ "${_node_value}" != template:* && "${_node_value}" != eval:* && "${_node_value}" != raw:* ]]; then
    abort "Non-templating text node was found: '$(trim "${_node_value}")'"
  fi
  _mode="${_node_value%%:*}"
  _body="${_node_value#*:}"
  if [[ "${_body}" == object:* || "${_body}" == array:* || "${_body}" == string:* || "${_body}" == number:* || "${_body}" == boolean:* ]] \
    ; then
    if [[ "${_body}" != string:* ]]; then
      _quote="no"
    fi
    _expected_type="${_body%%:*}"
    _body="${_body#*:}"
  fi

  if [[ "${_mode}" == "template" || "${_mode}" == "eval" ]]; then
    function _err_handler() {
      abort "Failed on eval: 'echo \"${_body}\"'"
    }
    local _error_prefix="ERROR: " _error_file _error_out _original_err_handler _content_file _node_value_file
    export _path
    _original_err_handler="$(trap -p ERR)"
    trap _err_handler ERR
    _node_value_file="$(_templating_files_dir)/templating.nodevalue"
    _node_path_file="$(_templating_files_dir)/templating.path"
    _content_file="$(_templating_files_dir)/templating.content"
    _error_file="$(_templating_files_dir)/templating.stderr"
    rm -f "${_error_file}" "${_node_value_file}" "${_node_path_file}" "${_content_file}"
    touch "${_error_file}" "${_node_value_file}" "${_node_path_file}" "${_content_file}"
    echo "${_node_value}" > "${_node_value_file}"
    echo "${_path}" > "${_node_path_file}"
    echo "${_self_content}" > "${_content_file}"
    debug "error: '${_error_file}'"
    # Perform the 'templating'
    _ret="$(eval "echo ${_body}" 2>"${_error_file}")"
    [[ -z "${_original_err_handler}" ]] || ${_original_err_handler}
    unset _path
    # shellcheck disable=SC2002
    _error_out="$(cat "${_error_file}")"
    [[ "${_error_out}" != *"${_error_prefix}"* ]] ||
      abort "Error was detected during templating:\n$(_compose_error_message_for_render_text_node "${_node_path_file}"  "${_node_value_file}" "${_content_file}" "${_error_file}")"
    debug "value: '${_ret}', stderr during eval:'${_error_out}'"
  elif [[ "${_mode}" == "raw" ]]; then
    _ret="${_body}"
  fi
  if [[ "${_quote}" == yes ]]; then
    _ret="${_ret//\\/\\\\}"
    _ret="\"${_ret//\"/\\\"}\""
  fi
  local _actual_type="(malformed)"
  _actual_type="$(echo "${_ret}" | jq -r '.|type')" ||
    abort "'$(trim "${_node_value}")' was rendered into '${_ret}' and it seems not a wel-formed JSON.\n$(_compose_error_message_for_render_text_node "${_node_path_file}"  "${_node_value_file}" "${_content_file}" "${_error_file}")"
  debug "expected type:'${_expected_type}' actual type:'${_actual_type}'"
  [[ "${_expected_type}" == "${_actual_type}" ]] ||
    abort "Type mismatch was detected for:'$(trim "${_node_value}")' expected type:'${_expected_type}' actual type:'${_actual_type}'\n$(_compose_error_message_for_render_text_node "${_node_path_file}"  "${_node_value_file}" "${_content_file}" "${_error_file}")"
  echo "${_ret}"
  return "${_ret_code}"
}

function _compose_error_message_for_render_text_node() {
  local _nodepath_file="${1}" _nodevalue_file="${2}"  _content_file="${3}" _error_file="${4}"
  printf "    nodepath: '%s'
    nodevalue: '%s'
    content: '%s'
    inherited files:
    %s
    error: '%s'
    " "$(cat "${_nodepath_file}")" "$(cat "${_nodevalue_file}")" "$(cat "${_content_file}")" "$(_misctemp_files_dir_nodepool_logfile_read)" "$(cat "${_error_file}")"
}

####
#
# Function intended to be used on 'script inheritance'.
# Search for a file from an environment variable "_JF_PATH" and prints an absolute path of it if found.
# In case it is not found, 1 will be returned.
#
# This function is indirectly called by 'jsonize' function through a command line it constructs
#
function find_file() {
  local _target="${1}"
  search_file_in "${_target}" "${_path}"
}

function _source_files() {
  local _i _files
  # We know that there is no alphanumerically named files under $(_source_files_dir)
  # shellcheck disable=SC2012
  mapfile -t _files <<<"$(ls "$(_sourced_files_dir)" | sort -n)"
  for _i in "${_files[@]}"; do
    [[ "${_i}" == "" ]] && break
    # shellcheck disable=SC1090
    source "$(_sourced_files_dir)/${_i}"
  done
}

function _define_builtin_functions() {
  ####
  #
  # This is a function intended to be used on templating (_render_text_node)
  function ref() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    local _path="${1}"
    local value type
    info "args:'${*}'"
    value=$(value_at "$(safe_path "${_path}")" "$(self)")
    type="$(type_of "${value}")"
    debug "value:'${value}'(type:'${type}') path:'${_path}'"
    if [[ "${type}" == string && ("${value}" == eval:* || "${value}" == template:*) ]]; then
      local ret
      _check_cyclic_dependency "${_path}" reference
      ret="$(_render_text_node "${value}" "${_path}" "${_self_content}")" || abort "TODO"
      _unmark_as_in_progress "${_path}" reference
      jq -r -c '.' <(echo "${ret}")
    else
      echo "${value}"
    fi
  }

  function refexists() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    local _path="${1}"
    info "args:'${*}'"
    has_value_at "$(safe_path "${_path}")" "$(self)"
  }

  function reftag() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    local _tagname="${1}"
    local _curpath
    info "args:'${*}'"
    _curpath="$(curn)"
    while [[ "${_curpath}" != "" ]]; do
      _curpath="$(parent "${_curpath}")"
      if refexists "${_curpath}.${_tagname}"; then
        ref "${_curpath}.${_tagname}"
        return 0
      fi
    done
    abort "The specified tag:'${_tagname}' was not found from the current path:'$(curn)'"
  }

  ####
  # Prints the entire file (before templating).
  #
  # This is a function intended to be used on templating (_render_text_node)
  function self() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    info "args:(none)"
    echo "${_self_content}"
  }
  ####
  # A function that prints a node path to the text node, where the calls this function is directly made.
  function curn() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    info "args:(none)"
    debug "cur:_path='${_path}'"
    echo "${_path}"
  }

  ####
  # A function that prints a node path to a container element, which is an array or an object, that encloses the entry
  # belongs to.
  # An entry here means a pair of key and value or an element in an array.
  function cur() {
    info "args:'${*}'"
    parent "$(curn)"
  }

  ####
  # Prints a path to a parent node of a given path.
  # Note that single level path results in an empty string.
  # That is ```parent .node``` will print nothing.
  # This is intentional to be able to do ```$(ref $(parent $(cur)).uncle)``` even if ```$(cur)``` is either
  # ```.node``` or ```.node.child```
  function parent() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    local _path="${1}" _level="${2:-1}"
    local _err
    info "args:'${*}'"
    if [[ ! "${_path}" == .* ]]; then
      abort "Path was not valid:(${_path}), it must start with a '.'"
    fi
    if [[ "${_path}" == "." ]]; then
      abort "Root does not have a parent.:(${_path})"
    fi
    _err="$(mktemp)"
    jq -r -c -n -L "${JF_BASEDIR}/lib" \
      'import "shared" as shared;
      .|path('"${_path}"')|.[0:-'"${_level}"']|shared::path2pexp(.)' 2>"${_err}" ||
      abort "$(printf \
        "Error was reported for node path:'${_path}' and level:'${_level}' by jq command. Forgot quoting?:\n%s" \
        "$(cat "${_err}")")"
  }

  function array_append() {
    # shellcheck disable=SC2181
    [[ $? == 0 ]] || abort "Preceding failure was detected by '${FUNCNAME[0]}'."
    [[ $# -gt 0 ]] || abort "No argument was given to built-in:'array_append'"
    local _cur="${1}"
    local _i
    shift
    info "args:'${*}'"
    for _i in "${@}"; do
      _cur="$(_array_append "${_cur}" "${_i}")"
    done
    echo "${_cur}"
  }

  function _array_append() {
    info "jq -r -c -n"
    info "--argjson a '${1}'"
    info "--argjson b '${2}'"
    # shellcheck disable=SC2154
    info "'\$a + \$b'"
    jq -r -c -n --argjson a "${1}" --argjson b "${2}" '$a + $b'
  }

  function error() {
    abort "${@}"
  }
}
