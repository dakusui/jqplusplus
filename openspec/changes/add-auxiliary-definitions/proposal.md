## Why

JSON++ authors sometimes need values while evaluating computed keys or values that must not appear in the generated configuration. Today they must place those values in the ordinary output tree, while `yq++` attempts to hide them by deleting every `_`-prefixed field and therefore silently deletes legitimate data; issue #108 supersedes the narrower symptom recorded by #67.

## What Changes

- Introduce an **auxiliary definition**, a named value under a root `$aux` object that participates in composition and expression evaluation but is omitted from the evaluation result.
- Compose `$aux` contributions with the same `$extends` and `$includes` precedence and value-composition rules as ordinary object fields.
- Bind the composed auxiliary object to the `$aux` jq variable during both key-side and value-side expression evaluation, and process `eval:` and `raw:` keys and values within the auxiliary object.
- Keep `$local` exclusively for local-node definitions; this change does not alter or rename it.
- Make `yq++` preserve ordinary `_`-prefixed keys so that JSON and YAML front-ends have the same JSON data-model semantics.
- Preserve `raw:$aux` as the escape for producing a literal output key named `$aux`.
- Document the project's pre-1.0 compatibility policy: while the project remains at `0.0.x` with limited adoption, an incompatible change does not require a deprecation period, dual syntax, automated migration, or rollback procedure unless the change explicitly requires one.
- **BREAKING** Reserve root `$aux` as JSON++ control syntax; an existing literal root key with that name must use `raw:$aux`.
- **BREAKING** Stop `yq++` from automatically removing `_`-prefixed holder fields; fields previously hidden by that convention will appear in output.

## Capabilities

### New Capabilities

- `expression-evaluation/auxiliary-definitions`: Declaring, composing, evaluating, referencing, omitting, and diagnosing auxiliary definitions, including consistent `_`-prefixed field preservation across output formats.

### Modified Capabilities

None.

## Impact

- Expression and invocation context in `internal/`, including the state carried through structural composition and the variables supplied to key-side and value-side jq evaluation.
- `tools/bin/yq++`, which will become a format-conversion wrapper rather than applying a field-name convention.
- Unit tests and end-to-end cases for composition precedence, computed keys and values, escaping, diagnostics, cycles, and JSON/YAML parity.
- The README, design principles, syntax reference, evaluation model, builtin reference, terminology, compatibility notes, and changelog.
