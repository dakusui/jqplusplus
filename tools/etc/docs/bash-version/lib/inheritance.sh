[[ "${_INHERITANCE_SH:-""}" == "yes" ]] && return 0
_INHERITANCE_SH=yes

function expand_inheritances() {
  local _nodeentry="${1}" _validation_mode="${2}" _jf_path="${3}"
  local _jsonized_content _out _absfile
  local -a _specifier
  perf "begin: ${_nodeentry}"
  touch "$(_misctemp_files_dir_nodepool_logfile)"

  mapfile -d ';' -t _specifier <<<"$(_normalize_nodeentry "${_nodeentry}" "${_jf_path}")"
  _absfile="$(search_file_in "${_specifier[0]}" "${_jf_path}")"
  _jsonized_content="$(jsonize "${_absfile}" "${_specifier[1]}" "$(join_by ';' "${_specifier[@]:2}")")"
  # Fail on command substitution cannot be checked directly
  # shellcheck disable=SC2181
  [[ $? == 0 ]] || {
    error "Failed to convert a file:'${_absfile}' into to a json."
    return 1
  }
  validate_jf_json "${_jsonized_content}" "${_validation_mode}"
  if is_object "${_jsonized_content}"; then
    local _local_nodes_dir _c _extends_expanded
    ####
    # Strangely the line above does not causes a quit on a failure.
    # Explitly check and abotrt this functino.
    _c="$(expand_filelevel_inheritances "${_absfile}" "${_jsonized_content}" "${_validation_mode}" "$(dirname "${_absfile}"):${_jf_path}")" ||
      abort "File-level expansion failed for '${_nodeentry}'\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
    debug "_nodeentry='${_nodeentry}', _absfile='${_absfile}'"
    _local_nodes_dir=$(materialize_local_nodes "${_absfile}" "${_c}")
    expand_inheritances_for_local_nodes "${_local_nodes_dir}" "${_jf_path}"
    _extends_expanded="$(expand_nodelevel_inheritances "${_c}" \
      "${_validation_mode}" \
      "${_local_nodes_dir}:$(dirname "${_absfile}"):${_jf_path}")" ||
      abort_no_stacktrace "Failed to expand node level inheritance for '${_nodeentry}'(3)\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
    _out="${_extends_expanded}"
  else
    : # Clear $?
    _out="${_jsonized_content}"
  fi
  echo "${_out}"
  perf "end: ${_nodeentry}"
}

function expand_filelevel_inheritances() {
  local _absfile="${1}" _content="${2}" _validation_mode="${3}" _path="${4}"
  ####
  # This is intentionally using single quotes to pass quoted path expression to jq.
  # shellcheck disable=SC2016
  local _cur
  perf "begin"
  is_debug_enabled && debug "content='${_content}'"
  _cur="${_content}"
  local -a _parents
  # BEGIN: normal filelevel
  # shellcheck disable=SC2016
  mapfile -t _parents <<<"$(value_at '."$extends"' "${_content}" '[]' | jq -c -r '.[]')"
  if ! is_effectively_empty_array "${_parents[@]}"; then
    local i
    for i in "${_parents[@]}"; do
      local _c _parent
      echo "extends:<${i}>" >>"$(_misctemp_files_dir_nodepool_logfile)"
      _parent="$(nodepool_read_nodeentry "${i}" "${_validation_mode}" "${_path}")"
      [[ "${_parent}" == "" ]] && return 1
      _c="$(merge_object_nodes "${_parent}" "${_cur}")"
      # Cannot check the exit code directly because of command substitution
      # shellcheck disable=SC2181
      [[ $? == 0 ]] || abort "Failed to merge file:'${i}' with content:'$(trim "${_cur}")'\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
      _cur="${_c}"
    done
  fi
  # END: normal filelevel
  # BEGIN: reverse filelevel
  # shellcheck disable=SC2016
  mapfile -t _children <<<"$(value_at '."$includes"' "${_content}" '[]' | jq -c -r '.[]')"
  if ! is_effectively_empty_array "${_children[@]}"; then
    local i
    for i in "${_children[@]}"; do
      local _c _child
      echo "includes:<${i}>" >>"$(_misctemp_files_dir_nodepool_logfile)"
      _child="$(nodepool_read_nodeentry "${i}" "${_validation_mode}" "${_path}")"
      _c="$(merge_object_nodes "${_cur}" "${_child}")"
      # Cannot check the exit code directly because of command substitution
      # shellcheck disable=SC2181
      [[ $? == 0 ]] || abort "Failed to merge file:'${i}' with content:'$(trim "${_cur}")'\ninherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
      _cur="${_c}"
    done
  fi
  # END: reverse filelevel
  # remove reserved keywords
  echo "${_cur}" | jq -r -c '.|del(.["$extends"])' | jq -r -c '.|del(.["$includes"])'
  perf "end"
}

function expand_inheritances_for_local_nodes() {
  local _local_nodes_dir="${1}" _jf_path="${2}"
  debug "begin: _local_nodes_dir=${_local_nodes_dir}"
  while IFS= read -r -d '' i; do
    local _f="${i}"
    debug "expanding inheritance of a local node:'${i}'"
    #                                                 nodeentry
    #                                                 |        validation_mode
    #                                                 |        |     path
    #                                                 |        |     |
    mktemp_with_content "$(nodepool_read_nodeentry """${_f}""" "no" "${_local_nodes_dir}:${_jf_path}")" ".local.json" >"${_f}"
    debug "...expanded"
  done < <(find "${_local_nodes_dir}" -maxdepth 1 -type f -print0)
  debug "end"
}

function expand_nodelevel_inheritances() {
  local _content="${1}" _validation_mode="${2}" _path="${3}"
  local _extends_expanded _includes_expanded _clean _content _ret
  perf "begin"
  _clean="${_content}"
  _clean="$(remove_nodes "${_clean}" '$extends')"
  _clean="$(remove_nodes "${_clean}" '$includes')"
  _extends_expanded="$(_expand_nodelevel_inheritances "${_content}" "${_validation_mode}" "${_path}" '$extends')" ||
    abort "Failed to expand node level inheritance for node:'$(trim "${_content}")'(1)\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
  _extends_expanded=$(merge_object_nodes "${_extends_expanded}" "${_clean}") ||
    abort "Failed to expand node level inheritance for node:'$(trim "${_content}")'(2)\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
  _includes_expanded="$(_expand_nodelevel_inheritances "${_content}" "${_validation_mode}" "${_path}" '$includes')" ||
    abort "Failed to expand node level inheritance for node:'$(trim "${_content}")'(3)\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
  _ret=$(merge_object_nodes "${_extends_expanded}" "${_includes_expanded}") ||
    abort "Failed to expand node level inheritance for node:'$(trim "${_content}")'(4)\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
  _ret="$(remove_nodes "${_ret}" '$extends')"
  _ret="$(remove_nodes "${_ret}" '$includes')"
  echo "${_ret}"
  perf "end"
}

function _expand_nodelevel_inheritances() {
  local _content="${1}" _validation_mode="${2}" _path="${3}" _keyword="${4}"
  local _cur='{}' i
  local -a _keys
  perf "begin"
  is_debug_enabled && debug "_content='${_content}'"
  # Intentional single quote to find a keyword that starts with '$'
  mapfile -t _keys < <(paths_of "${_content}" "${_keyword}")
  is_effectively_empty_array "${_keys[@]}" && _keys=()
  for i in "${_keys[@]}"; do
    local _jj _p="${i%.\"${_keyword}\"}"
    local -a _extendeds
    mapfile -t _extendeds < <(echo "${_content}" | jq -r -c "${i}[]")
    is_effectively_empty_array "${_extendeds[@]}" && _extendeds=()
    for _jj in "${_extendeds[@]}"; do
      local _tmp_content
      debug "processing nodeentry: '${_jj}'"
      echo "local:<${i}>:<${_jj}>" >>"$(_misctemp_files_dir_nodepool_logfile)"
      local _merged_piece_content
      if has_value_at "${_p}" "${_cur}"; then
        local _cur_piece _next_piece
        _cur_piece="$(echo "${_cur}" | jq -r -c "${_p}")"
        _next_piece="$(nodepool_read_nodeentry "${_jj}" "${_validation_mode}" "${_path}")"
        if [[ "${_keyword}" == '$extends' ]]; then
          _merged_piece_content="$(merge_object_nodes "${_next_piece}" "${_cur_piece}")"
        elif [[ "${_keyword}" == '$includes' ]]; then
          _merged_piece_content="$(merge_object_nodes "${_cur_piece}" "${_next_piece}")"
        else
          abort "Unknown keyword: '${_keyword}' was specified.\nInherited files:\n$(_misctemp_files_dir_nodepool_logfile_read)"
        fi
        # shellcheck disable=SC2181
        [[ $? == 0 ]] || abort_no_stacktrace "Failed to merge node:'$(trim "${_cur}")' with _nodeentry:'${_jj}'\nInherited files:\n" "$(_misctemp_files_dir_nodepool_logfile_read)"
      else
        local _expanded_tmp
        _expanded_tmp="$(nodepool_read_nodeentry "${_jj}" "${_validation_mode}" "${_path}")"
        # shellcheck disable=SC2181
        [[ $? == 0 ]] || abort_no_stacktrace "Failed to expand inheritances for '${_jj}'\nInherited files:\n" "$(_misctemp_files_dir_nodepool_logfile_read)"
        _merged_piece_content="${_expanded_tmp}"
      fi
      is_debug_enabled && debug "_merged_piece_content:'${_merged_piece_content}'"
      _tmp_content="$(jq -n "input | ${_p}=input" <(echo "${_cur}") <(echo "${_merged_piece_content}"))"
      _cur="${_tmp_content}"
      is_debug_enabled && debug "_cur(updated):'${_cur}'"
    done
  done
  perf "end"
  echo "${_cur}" | jq -r -c .
}

# "
function materialize_local_nodes() {
  local _absfile="${1}" _content="${2}"
  local _ret _i
  debug "begin"
  _ret="$(mk_localnodedir "${_absfile}")"
  # Quickfix for Issue #98: Probably we should filter null, which can be produced by the first predicate (."$local")
  mapfile -t _local_nodes < <(echo "${_content}" | jq -r -c '."$local"
                                   |. as $local
                                   |keys[]
                                   |. as $k
                                   |$local|getpath([$k])
                                          |. as $v
                                          |[$k, $v]' 2>/dev/null)
  for _i in "${_local_nodes[@]}"; do
    local _f _c
    _f="$(echo "${_i}" | jq -r -c '.[0]')"
    _c="$(echo "${_i}" | jq -c '.[1]')"
    debug "_i:'${_i}'"
    debug "_f:'${_f}'"
    debug "_c:'${_c}'"
    debug "_out:${_ret}/${_f}"
    echo "${_c}" >"${_ret}/${_f}"
  done
  debug "_ret:${_ret}"
  echo "${_ret}"
  debug "end"
}
