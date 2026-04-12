#!/usr/bin/env bash
set -E -eu -o pipefail
# shopt -s inherit_errexit

function extract_style() {
  cat "../../../docs/_style.html" | xmllint --nocdata --html --xpath '//style' - | sed -E 's/^(\]\]>)?<\/?style>(<\!\[CDATA\[)?$//g'
}

# Normalize a filestem (e.g. "concepts/overview") to a safe HTML id (e.g. "concepts-overview").
function filestem_to_id() {
  echo "${1//\//-}"
}

function extract_content() {
  local _filestem="${1}"
  local _id
  _id=$(filestem_to_id "${_filestem}")
  cat "../../../docs/${_filestem}.html" | xmllint --nocdata --html --xpath '//div[@id="content"]' - | sed -E 's/<div id="content">/<div id="'"${_id}"'_content">/g'
}

function copy_images() {
  cp ./images/* ../../../docs/images/
}

function render_style() {
  extract_style
  cat <<STYLE
// For tabs
body {font-family: Arial;}

/* Style the tab */
.tab {
  overflow: hidden;
  border: 1px solid #ccc;
  background-color: #f1f1f1;
}

/* Style the buttons inside the tab */
.tab button {
  background-color: inherit;
  float: left;
  border: none;
  outline: none;
  cursor: pointer;
  padding: 14px 16px;
  transition: 0.3s;
  font-size: 17px;
}

.tab button .tab-folder {
  display: block;
  font-size: 12px;
  line-height: 1.1;
  color: #555;
}

.tab button .tab-page {
  display: block;
  line-height: 1.2;
}

/* Change background color of buttons on hover */
.tab button:hover {
  background-color: #ddd;
}

/* Create an active/current tablink class */
.tab button.active {
  background-color: #ccc;
}

/* Style the tab content */
.tabcontent {
  display: none;
  padding: 6px 12px;
  border: 1px solid #ccc;
  border-top: none;
}
STYLE
}

function format_button_label() {
  local _filestem="${1}"
  local _include_parent="${2:-false}"
  local _label
  _label=$(basename "${_filestem}")
  if [[ "${_include_parent}" != "true" ]]; then
    echo '<span class="tab-folder">&nbsp;</span><span class="tab-page">'"${_label}"'</span>'
    return
  fi
  local _parent
  _parent=$(dirname "${_filestem}")
  if [[ "${_parent}" == "." || "${_parent}" == "/" ]]; then
    echo '<span class="tab-folder">&nbsp;</span><span class="tab-page">'"${_label}"'</span>'
    return
  fi
  echo '<span class="tab-folder">'"${_parent}/"'</span><span class="tab-page">'"${_label}"'</span>'
}

function render_default_button() {
  local _filestem="${1}"
  local _include_parent="${2:-false}"
  local _id
  _id=$(filestem_to_id "${_filestem}")
  local _label
  _label=$(format_button_label "${_filestem}" "${_include_parent}")
  echo '<button class="tablinks" onclick="openTab(event, '"'${_id}'"')" id="defaultOpen">'"${_label}"'</button>'
}

function render_button() {
  local _filestem="${1}"
  local _include_parent="${2:-false}"
  local _id
  _id=$(filestem_to_id "${_filestem}")
  local _label
  _label=$(format_button_label "${_filestem}" "${_include_parent}")
  echo '<button class="tablinks" onclick="openTab(event, '"'${_id}'"')">'"${_label}"'</button>'
}

function render_github_button() {
  cat <<ELEM
<button class="tablinks" onclick="location.href='https://github.com/dakusui/jqplusplus';">
  <img src="images/github.png" width="60px;20px" onclick="location.href = 'https://github.com/dakusui/jqplusplus';" align="middle"/>
</button>
ELEM
}

function render_all_buttons() {
  local _top_filestem="${1}"
  local _top_id
  _top_id=$(filestem_to_id "${_top_filestem}")
  local _previous_dir
  _previous_dir=$(dirname "${_top_filestem}")
  shift
  local _filestems=("$@")
  echo '<div class="tab">'
  local i
  render_github_button
  render_default_button "${_top_filestem}" "true"
  for i in "${_filestems[@]}"; do
    if [[ "$(filestem_to_id "${i}")" == "${_top_id}" ]]; then
      continue
    fi
    local _current_dir
    _current_dir=$(dirname "${i}")
    if [[ "${_current_dir}" != "${_previous_dir}" ]]; then
      render_button "${i}" "true"
    else
      render_button "${i}" "false"
    fi
    _previous_dir="${_current_dir}"
  done
  echo '</div>'
}

function render_content() {
  local _filestem="${1}"
  local _id
  _id=$(filestem_to_id "${_filestem}")
  echo '<div id="'${_id}'" class="tabcontent">'
  extract_content "${_filestem}"
  echo '<div class="paragraph text-right"><p>'
  echo '<a href="'${_filestem}'.html">open this page in a window</a>'
  echo '</p></div>'
  echo '</div>'
}

function render_all_contents() {
  local _filestems=("$@")
  for i in "${_filestems[@]}"; do
    render_content "${i}"
  done
}

function begin_header() {
  cat <<BEGINHEADER
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<meta name="generator" content="Asciidoctor 2.0.10">
<title>jq++: JSON with structural reuse and expression evaluation</title>
<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Open+Sans:300,300italic,400,400italic,600,600italic%7CNoto+Serif:400,400italic,700,700italic%7CDroid+Sans+Mono:400,700">
<style>
BEGINHEADER
}

function end_header() {
  echo '</style>'
  echo '</head>'
}

function begin_body() {
  echo '<body>'
}

function end_body() {
  cat <<FOOTER
<script>
function openTab(evt, cityName) {
  var i, tabcontent, tablinks;
  tabcontent = document.getElementsByClassName("tabcontent");
  for (i = 0; i < tabcontent.length; i++) {
    tabcontent[i].style.display = "none";
  }
  tablinks = document.getElementsByClassName("tablinks");
  for (i = 0; i < tablinks.length; i++) {
    tablinks[i].className = tablinks[i].className.replace(" active", "");
  }
  document.getElementById(cityName).style.display = "block";
  evt.currentTarget.className += " active";
}

// Get the element with id="defaultOpen" and click on it
document.getElementById("defaultOpen").click();

</script>
</body>
FOOTER
}

function render_footer() {
  echo '</html>'
}

function render() {
  local _top="${1}"
  shift
  local _filestems=("$@")
  begin_header
  render_style
  end_header
  begin_body
  # shellcheck disable=SC2068
  render_all_buttons "${_top}" "${_filestems[@]}"
  # shellcheck disable=SC2068
  render_all_contents "${_filestems[@]}"
  end_body
  render_footer
  copy_images
}

function main() {
  local _top='concepts/overview'
  local _targets
  mapfile -t _targets < <(find . -name '*.adoc' | grep -v '_style' | sed 's|^\./||' | sed 's|\.adoc$||' | sort)
  render "${_top}" "${_targets[@]}"
}

cd "$(dirname ${0})"
main
