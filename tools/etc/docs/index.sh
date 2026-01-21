#!/usr/bin/env bash
set -E -eu -o pipefail
# shopt -s inherit_errexit

function extract_style() {
  cat "../../../docs/_style.html" | xmllint --nocdata --html --xpath '//style' - | sed -E 's/^(\]\]>)?<\/?style>(<\!\[CDATA\[)?$//g'
}

function extract_content() {
  local _filestem="${1}"
  cat "../../../docs/${_filestem}.html" | xmllint --nocdata --html --xpath '//div[@id="content"]' - | sed -E 's/<div id="content">/<div id="'"${_filestem}"'_content">/g'
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

function render_default_button() {
  local _filestem="${1}"
  echo '<button class="tablinks" onclick="openTab(event, '"'${_filestem}'"')" id="defaultOpen">'"${_filestem}"'</button>'
}

function render_button() {
  local _filestem="${1}"
  echo '<button class="tablinks" onclick="openTab(event, '"'${_filestem}'"')">'"${_filestem}"'</button>'
}

function render_github_button() {
  cat <<ELEM
<button class="tablinks" onclick="location.href='https://github.com/dakusui/jq-front';">
  <img src="images/github.png" width="60px;20px" onclick="location.href = 'https://github.com/dakusui/jq-front';" align="middle"/>
</button>
ELEM
}

function render_all_buttons() {
  local _top_filestem="${1}"
  shift
  local _filestems=("$@")
  echo '<div class="tab">'
  local i
  render_github_button
  render_default_button "${_top_filestem}"
  for i in "${_filestems[@]}"; do
    if [[ "${i}" == "${_top_filestem}" ]]; then
      continue
    fi
    render_button "${i}"
  done
  echo '</div>'
}

function render_content() {
  local _filestem="${1}"
  echo '<div id="'${_filestem}'" class="tabcontent">'
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
<title>jq-front: JSON with inheritance and templating</title>
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
  local _filestems=("$@")
  begin_header
  render_style
  end_header
  begin_body
  # shellcheck disable=SC2068
  render_all_buttons top ${_targets[@]}
  # shellcheck disable=SC2068
  render_all_contents ${_targets[@]}
  end_body
  render_footer
  copy_images
}

function main() {
  local _targets
  _targets=$(ls *.adoc | sed -E 's/\.adoc$//' | grep -v _style | sort)
  # shellcheck disable=SC2068
  render ${_targets[@]}
}

cd "$(dirname ${0})"
main
