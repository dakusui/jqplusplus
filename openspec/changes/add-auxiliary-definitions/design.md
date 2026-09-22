## Context

See `proposal.md` for the motivation and issue #108 for the pain. The observable contract is in `specs/expression-evaluation/auxiliary-definitions/spec.md`.

Today `LoadAndResolveInheritancesRecursively` materializes a source file's `$local` entries and immediately deletes `$local` before file-level or node-level structural composition. `NodeEntryValue` then carries only the resolved ordinary object and accumulated jq modules. Key-side evaluation runs over that object with `$cur`; value-side evaluation subsequently installs the builtin variables and functions and evaluates `eval:` and `raw:` strings until stable.

`yq++` adds a second, format-specific semantic step after `jq++`: a recursive jq filter deletes every `_`-prefixed key before YAML emission. The engine cannot distinguish a holder field from legitimate data because the convention has no explicit declaration.

The design must preserve three constraints:

- `$local` entries are destination-relative local-node templates. Evaluating them in place would change `$cur`, `parent`, and `reftag` behavior and would make unused templates fail eagerly.
- `raw:$aux` must still be able to create a literal output key named `$aux`; omission cannot be implemented as an unconditional final deletion by spelling.
- Auxiliary values used by ordinary computed keys must be ready before the existing ordinary key-side phase begins.

## Goals / Non-Goals

**Goals:**

- Represent the ordinary result and the composed auxiliary namespace as separate engine state.
- Reuse the existing structural-composition rules for auxiliary contributions.
- Prepare auxiliary computed keys and values before ordinary key-side evaluation.
- Supply one composed `$aux` jq variable consistently to ordinary key-side and value-side expressions.
- Preserve existing reference and cycle behavior while auxiliary values are prepared.
- Remove format-specific underscore semantics from `yq++`.

**Non-Goals:**

- Renaming or changing `$local`.
- Providing file-lexical, access-controlled, confidential, or source-private values.
- Hiding arbitrary ordinary fields by prefix, pattern, configuration, or command-line option.
- Generalizing builtin availability on key-side expressions beyond the `$aux` variable required here.
- Binding an inherited value consumed at a particular merge site for arbitrary jq processing.
- Defining a second auxiliary namespace at nested object positions.

## Decisions

### 1. Carry `$aux` out of band from the ordinary result

Extend the state returned for a resolved node with an auxiliary object alongside the ordinary object and jq modules. At file read time, validate and extract a root `$aux` object before ordinary structural and expression walkers receive the object. `$local` materialization and removal remain on their existing path.

Conceptually, resolved state becomes:

```text
NodeEntryValue
  ordinary object
  auxiliary object
  jq modules
```

The auxiliary object is not an ordinary output key even though authors declare it with the root `$aux` control key. This identity prevents an escaped literal `$aux` produced later by `raw:$aux` from being mistaken for the control namespace.

**Alternative: keep `$aux` in the ordinary map and delete it before serialization.** Rejected because `raw:$aux` or a computed key can intentionally create the same spelling. A name-based final deletion cannot distinguish the control object from literal output and would violate the escape guarantee.

### 2. Compose auxiliary objects in parallel with ordinary objects

Whenever structural composition combines two node results, combine their auxiliary objects with the same merge direction and `mergeObjects` rules used for their ordinary objects. This gives `$extends`, `$includes`, recursive object composition, array composition, and overrides the same precedence on both sides.

The auxiliary namespace is global to the completed invocation. Every source reached through structural composition contributes to it in the same deterministic traversal that currently accumulates jq modules. Expressions do not retain source-file provenance; after composition they all receive the final auxiliary object.

Ground array-composition markers in both the ordinary and auxiliary objects before expression evaluation so an unresolved marker cannot disappear merely because it is auxiliary.

**Alternative: entry-document-only scope.** Rejected because a reusable parent could not carry the auxiliary values its inherited expressions need.

**Alternative: lexical source-file scope.** Rejected because merged JSON values currently discard source provenance. Preserving an environment on every inherited expression would introduce a new scoping system and would prevent the intended child override behavior.

### 3. Prepare the auxiliary namespace before ordinary key evaluation

After structural composition completes:

```text
auxiliary key-side evaluation
        |
auxiliary value-side evaluation
        |
ordinary key-side evaluation with $aux
        |
ordinary value-side evaluation with $aux
        |
serialize ordinary object
```

Auxiliary preparation uses the existing key-before-value rule. It is eager: an invalid unused auxiliary expression is an error, just as an invalid ordinary output expression is today. Finishing auxiliary value evaluation before ordinary key evaluation ensures a computed auxiliary array can safely drive an ordinary computed key.

Auxiliary expressions evaluate against a synthetic read view containing the structurally composed ordinary object plus the current auxiliary object at the control path `$aux`. The synthetic view exists only during auxiliary preparation and is never serialized. It lets existing value-side `ref(["$aux", ...])` lazily resolve another auxiliary expression and reuse current cycle detection.

Direct `$aux.name` access follows jq's ordinary object semantics. During auxiliary preparation, authors who need lazy evaluation of another computed auxiliary definition use `ref(["$aux", "name"])`, just as ordinary computed fields use `ref` rather than direct dot access when evaluation is required. Once preparation finishes, ordinary expressions receive processed auxiliary values through `$aux`.

**Alternative: process auxiliary and ordinary expressions in the same value-side batch.** Rejected because an ordinary expression could see another auxiliary definition's unevaluated `eval:` string. Deferring type errors or imposing map-iteration order would make results unstable.

### 4. Adapt evaluation walkers to separate target and read context

The existing key and value processors both use the object being mutated as the jq input. Auxiliary preparation needs to mutate the auxiliary object while reading the synthetic combined view. Refactor the processors so the target being walked, the path prefix used for `$cur`, and the expression read context can be supplied separately.

For auxiliary targets, prefix current paths with `"$aux"`. Therefore an expression stored at auxiliary path `service.port` receives:

```text
$cur     = ["$aux", "service", "port"]
$curexpr = the equivalent path expression
```

Refresh the synthetic read view after each recursive key-side or value-side pass so newly computed auxiliary keys and values are visible on subsequent passes under the same fixed iteration limit used by ordinary evaluation.

For ordinary targets, preserve existing `$cur`, `$curexpr`, input-root, and recursion behavior; add only the prepared `$aux` variable to both key-side and value-side invocation specifications.

### 5. Use a jq variable rather than a lookup function

Bind the prepared object as `$aux` in ordinary key-side and value-side expressions. This makes literal and fully prepared auxiliary values available through normal jq object access without creating `aux`, `refaux`, or `auxexpr` function families.

A missing property consequently yields jq's normal missing-object-field result, usually `null`; existing expected-type validation still reports a mismatch when the surrounding `eval:` annotation requires another kind.

**Alternative: a special builtin function.** Rejected because function installation currently differs between key-side and value-side evaluation, while variables already exist on both paths. A function would also need a separate choice between direct and lazy lookup semantics.

### 6. Keep `$local` on its current lifecycle

Do not retain `$local` through expression processing and do not expose it through `$aux`. Continue materializing local nodes before removing the control key. When a local node is used, its object copy reaches the ordinary result and its expressions evaluate at the destination path as today.

**Alternative: reuse `$local` for auxiliary values.** Rejected because local-node templates may contain destination-relative expressions. Treating `$local` as an ordinary hidden object would eagerly evaluate unused templates at paths under `$local`, alter path-sensitive behavior, and cause node-level structural walkers to revisit template contents.

### 7. Reserve `$aux`, not an access-control or lifetime term

Use `$aux` for the input control key and jq variable. An auxiliary definition supports evaluation but is not observable output. The name describes its computational role without promising OOP-style `private` or `protected` access, lexical `local` scope, or a lifetime property shared by other control constructs.

Rejected names include `$private` and `$protected` because the composed namespace is visible and overridable across structural contributions; `$transient` because `$local`, `$extends`, and `$includes` are also absent from output; and `$context` because it suggests externally supplied, read-only data rather than composable definitions.

### 8. Remove underscore stripping from `yq++`

Delete the recursive `_strip` jq program and feed `jq++` JSON output directly to the YAML emitter. Both commands then preserve the same data, and output omission is owned solely by the JSON++ engine's explicit control constructs.

Add dedicated command-level coverage for `yq++`; the current normal autotest executor invokes `jq++` and does not exercise the wrapper's post-processing behavior.

## Risks / Trade-offs

**A new reserved root key can consume existing data.** A document currently using literal `$aux` would change meaning. → Document this as a breaking change, preserve `raw:$aux`, and add coexistence coverage for control and escaped literal keys.

**Removing the underscore convention can expose former holder fields.** Existing `yq++` users may see fields they expected the wrapper to delete. → Ship `$aux` and underscore preservation atomically and provide a direct migration example.

**Eager auxiliary preparation can reject unused definitions.** An invalid auxiliary expression fails even when ordinary output does not reference it. → State this rule explicitly and test it; eager preparation is the cost of a stable, fully processed `$aux` variable for key-side use.

**A global composed namespace can create name collisions.** Structurally unrelated sources may choose the same auxiliary name. → Apply deterministic ordinary composition precedence and recommend nesting related definitions under a distinctive object name. Do not describe `$aux` as encapsulation.

**Synthetic read state could leak into output.** Reusing one mutable map for the synthetic root would risk serializing control data or deleting an escaped literal. → Keep ordinary and auxiliary objects separate and construct read views without transferring ownership.

**Evaluation refactoring can change existing paths or recursion.** Separating target and context touches both evaluation passes. → Characterize existing key/value, `ref`, `reftag`, `$cur`, `$curexpr`, and cycle behavior before refactoring, then run all unit and end-to-end tests.

## Migration Plan

1. Release `$aux` support and underscore preservation together.
2. Move each `_`-prefixed holder into the root `$aux` object, preserving any desired nested organization.
3. Replace direct holder references with `$aux` variable access; use `ref(["$aux", ...])` inside auxiliary value preparation when lazy resolution of another computed auxiliary definition is required.
4. Leave legitimate `_`-prefixed output fields unchanged; `yq++` will preserve them.
5. Rewrite an intended literal root `$aux` key as `raw:$aux`.

Rolling back to an older release requires restoring the former holder convention in affected inputs because the older evaluator does not recognize `$aux`. No automatic dual-syntax period is provided: retaining prefix-based deletion would continue the silent data-loss defect this change removes.
