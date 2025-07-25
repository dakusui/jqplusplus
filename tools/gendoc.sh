#!/bin/bash -eu

function message() {
  local IFS=" "
  local _o
  _o="${1}"
  shift
  echo -e "${_o}" "$*" >&2
}

function main() {
  while IFS= read -r -d '' i; do
    local _src_file="${i}"
    message -n "Processing '${_src_file}'"
    docker run --rm \
      --user "$(id -u):$(id -g)" \
      -v "$(pwd)":/documents/ \
      asciidoctor/docker-asciidoctor \
      asciidoctor -r asciidoctor-diagram -a toc=left "${i}" -o "${i%.adoc}.html"
    message "...done"
  done < <(find "docs" -type f -name '*.adoc' -print0)
  message -n "Generating 'docs/index.html'"
  docs/index.sh >docs/index.html
  message "...done"
}

main "$@"