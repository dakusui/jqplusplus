---
name: asciidoc
description: Writing, creating, converting, and editing AsciiDoc (.adoc) files. Use this skill whenever the user needs to create a new .adoc file, update or edit an existing .adoc file, add content to an .adoc file, or convert content from another format (such as Markdown) into AsciiDoc. Trigger any time the user mentions .adoc, AsciiDoc, or asks to touch any file with the .adoc extension.
---

# AsciiDoc Writing

## Scope and preservation rule

Apply these conventions to **new or added content only**.
Unless the user explicitly asks you to reformat or clean up existing content, leave existing text exactly as-is — even if it doesn't follow these conventions.
This rule exists to prevent unintended diff noise and to respect the author's original choices.

## Conventions for new/added content

### One sentence per line

Write one sentence per line.
This makes version-control diffs meaningful — each changed line corresponds to a single logical unit.

**Example (correct):**
```
This module handles authentication.
It supports both OAuth2 and API key flows.
See the configuration reference for details.
```

**Example (incorrect — multiple sentences on one line):**
```
This module handles authentication. It supports both OAuth2 and API key flows.
```

### Blank line before lists

Always insert a blank line before any list (bulleted, numbered, or description list).
Without it, AsciiDoc may render the list items as a continuation of the preceding paragraph rather than a proper list.

**Example (correct):**
```
The following formats are supported:

* JSON
* YAML
* TOML
```

**Example (incorrect — missing blank line):**
```
The following formats are supported:
* JSON
* YAML
* TOML
```

### Escape plus signs

Write `{plus}` instead of a bare `+` in regular text.
A bare `+` is the constrained monospace delimiter in AsciiDoc and will cause unexpected rendering when used as an arithmetic or concatenation symbol.

**Example (correct):**
```
The result of 1{plus}2 is 3.
Use the {plus} operator to concatenate strings.
```

**Example (incorrect):**
```
The result of 1+2 is 3.
```

Note: Inside literal code blocks (delimited by `----` or backtick passthrough), `+` does not need escaping.

### Attribute definitions at the top

Place all document-level attribute definitions at the top of the file, immediately after the document title and optional author/revision lines.
This keeps document metadata in one predictable location.

**Example:**
```adoc
= Document Title
Author Name
:toc:
:icons: font
:source-highlighter: rouge
:my-custom-attribute: value

First paragraph of content starts here.
```

## Conversion from other formats

When converting Markdown or other formats to AsciiDoc, apply all conventions above to the converted output.
Common mapping reminders:

| Markdown | AsciiDoc |
|---|---|
| `# Heading` | `= Heading` (doc title) / `== Heading` (section) |
| `` `code` `` | `` `code` `` (same) |
| `**bold**` | `*bold*` |
| `_italic_` | `_italic_` |
| `[text](url)` | `link:url[text]` |
| ` ```lang ` (fenced block) | `[source,lang]\n----\n...\n----` |
