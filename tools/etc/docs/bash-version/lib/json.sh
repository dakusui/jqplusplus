[[ "${_JSON_SH:-""}" == "yes" ]] && return 0
_JSON_SH=yes

source "$(dirname "${BASH_SOURCE[0]}")/core.sh"

# Remove paths which end with a given keyword from a JSON object and prints it.
function remove_nodes() {
  local _content="${1}" _keyword="${2}"
  if is_object "${_content}"; then
    local _keys i
    # Intentional single quote to find a keyword that starts with '$'
    # shellcheck disable=SC2016
    mapfile -t _keys < <(paths_of "${_content}" "${_keyword}") || _keys=()
    for i in "${_keys[@]}"; do
      local _next
      _next="$(echo "${_content}" | jq ".|del(${i})")"
      _content="${_next}"
    done
  fi
  echo "${_content}"
}

# List paths in a json object which end with a given keyword.
function paths_of() {
  local _content="${1}" _keyword="${2}"
  echo "${_content}" | jq -r -c -L "${JF_BASEDIR}/lib" '#---
import "shared" as shared;

[paths(..)|. as $p|.[-1]|select(tostring=="'"${_keyword}"'")|$p]
              |sort
              |sort_by(length)
              |.[]
              |shared::path2pexp(.)'
}

function validate_jf_json() {
  local _content="${1}" _mode="${2}"
  if [[ "${_mode}" == "no" ]]; then
    debug "validation skipped"
    return 0
  elif [[ "${_mode}" == "strict" ]]; then
    _validate_jf_json_with "$(mktemp_with_content "${_content}" ".validate-strict.json")" "strict"
  elif [[ "${_mode}" == "lenient" ]]; then
    _validate_jf_json_with "$(mktemp_with_content "${_content}" ".validate-lenient.json")" "lenient"
  else
    abort "Unknown validation mode:'${_mode}' was set."
  fi
  return $?
}

function safe_path() {
  local _path="${1}"
  if [[ "${_path}" != "."* ]]; then
    abort "Invalid path '${_path}' was given. A JSON path should start with a dot ('.')"
  fi
  _safe_path "${_path}"
}

function _safe_path() {
  function _safe_path_head() {
    local _text="${1}"
    echo "$_text" | sed -E 's/\.("([^"]+)"([^"^.]+)?|([^"^.]+))(.*$)/\1/'
  }

  function _safe_path_tail() {
    local _text="${1}"
    echo "$_text" | sed -E 's/\.("([^"]+)"([^"^.]+)?|([^"^.]+))(.*$)/\5/'
  }

  local _path="${1}"
  local _rest="${_path}"

  while [ "${_rest}" != "" ]; do
    _safe_path_component "$(_safe_path_head "${_rest}")"
    _rest=$(_safe_path_tail "${_rest}")
  done
}

function _safe_path_component() {
  local _path_component="${1}"
  if [[ "${_path_component}" == '"'*'"' ]]; then
    echo -n ."${_path_component}"
  elif [[ "${_path_component}" == '"'*'"['*']' ]]; then
    echo -n ."${_path_component}"
  elif [[ "${_path_component}" != '"'* && "${_path_component}" != *'"' ]]; then
    echo -n .'"'"${_path_component}"'"'
  else
    abort "Invalid path component '${_path_component}' was found."
  fi
}

function _validate_jf_json_with() {
  validate_json "${1}" "${_JF_SCHEMA_DIR}/${2}.json"
}

function validate_json() {
  local _in="${1}" _schema_file="${2}"
  local _out=
  {
    local _ret
    debug "validating: '${_in}'"
    _out=$(ajv validate -s "${_schema_file}" -d "${_in}" 2>&1)
    _ret=$?
    debug "...validated: '${_out}'"
    return "${_ret}"
  } || {
    abort "Validation by ajv for '${_in}' was failed:\n${_out}"
  }
}

function is_object() {
  local _json_content="${1}"
  local _ret
  is_debug_enabled && debug "_json_content:'${_json_content}'"
  _ret="$(echo "${_json_content}" | jq '.|if type=="object" then 0 else 1 end' 2>/dev/null)"
  [[ "${_ret}" == "" ]] &&
    abort "Probably jq wrote something other than numeric value from the content:'$(trim "${_json_content}")'."
  return "${_ret}"
}

function type_of() {
  local _json_content="${1}"
  jq -r -c '.|type' <(echo "${_json_content}") || echo "string"
}

function has_value_at() {
  has_value_at_strict "${@}"
}

function has_value_at_strict() {
  local _path="${1}"
  local _json="${2}"
  local _r
  _r="$(echo "${_json}" | jq -r -c '
  . as $c |
  null | try path('"${_path}"') catch error("not an exact path expression.") | . as $p |
  length as $l |
  $p | .[$l - 1] | . as $last |
  $p |
  if $l - 1 <= 0 then
    []
  else
    [limit($l -1; .[])]
  end | . as $q |
  if $q | length == 0 then
    $c | has($last)
  else
    $c | getpath($q) | type == "object" and has($last)
  end')" 2> /dev/null || abort "Failed to access path:'${_path}' json:'${_json}'"
  if [[ "${_r}" == 'true' ]]; then
    return 0
  elif [[ "${_r}" == 'false' ]]; then
    return 1
  fi
  abort "INTERNAL ERROR: Invalid value:'${_r}' produced by 'has_value_at_strict'(_path:'${_path}' json:'${_json}')"
}

function value_at() {
  value_at_strict "${@}"
}

function value_at_strict() {
  local _path="${1}"
  local _json="${2}"
  if ! has_value_at "${_path}" "${_json}"; then
    if [[ -z "${3+x}" ]]; then
      abort "Failed to access '${_path}' and default value for it was not given."
    fi
    echo "${3}"
    return 0
  fi
  echo "${_json}" |
  jq -r -c '. as $c |
            null | try path('"${_path}"') catch error("not an exact path expression.") | . as $p |
            length as $l |
            $p | .[$l - 1] | . as $last |
            $p |
            if $l - 1 <= 0 then
              []
            else
              [limit($l -1; .[])]
            end | . as $q |
            if $q | length == 0 then
              $c | getpath([$last])
            else
              $c | getpath($p)
            end' 2> /dev/null
}

# Latter overrides former
function merge_object_nodes() {
  local _a="${1}" _b="${2}"
  local _afile _bfile _error
  _error=$(mktemp)
  perf "begin"
  [[ "${_a}" != '' ]] || abort "An empty string was given as _a"
  [[ "${_b}" != '' ]] || abort "An empty string was given as _b"
  is_debug_enabled && debug "merging _a:'${_a}' and _b:'${_b}'"
  _afile="$(mktemp)"
  echo "${_a}" >"${_afile}"
  _bfile="$(mktemp)"
  echo "${_b}" >"${_bfile}"
  # shellcheck disable=SC2016
  jq -r -c -n --slurpfile a "${_afile}" --slurpfile b "${_bfile}" -L "${JF_BASEDIR}/lib" \
    'import "shared" as shared;

    def value_at($n; $p):
      $n | getpath($p);

    def setvalue_at($n; $p; $v):
      def type_of($v):
        $v | type;
      def _setvalue_at($n; $p; $v):
        $n | try setpath($p; $v)
             catch error("Failed to process node at path:<\($p|shared::path2pexp(.))>; the value:<\($v)>).");
      $n | if type_of($v)=="object" or type_of($v)=="array" then
             if type_of(value_at($n; $p))!="object" and type_of(value_at($n; $p)!="array") then
               _setvalue_at(.;$p; $v)
             else
               .
             end
           else
             _setvalue_at(.; $p; $v)
           end;

    def merge_objects($a; $b):
      $b | [paths(shared::scalars_or_empty)]
         | reduce .[] as $p ($a; setvalue_at(.; $p; value_at($b; $p)));

    merge_objects($a[0]; $b[0])' 2>"${_error}" || {
    abort "$(printf "jq-front: Failed to merge object nodes:\n    a=<%s>\n    b=<%s>\nERROR: %s)" \
      "$(jq -r -c -n "${_a}|." || echo "MALFORMED: ${_a}")" \
      "$(jq -r -c -n "${_b}|." || echo "MALFORMED: ${_b}")" \
      "$(cat "${_error}" || echo "UNAVAILABLE")")"
  }
  rm -f "${_bfile}" "${_afile}"
  perf "end"
}
