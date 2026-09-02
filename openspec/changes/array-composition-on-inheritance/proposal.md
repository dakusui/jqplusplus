## Why

When a JSON++ document inherits from another through `$extends` or `$includes`, an object at a given key is merged with the inherited object, but an array at a given key replaces the inherited array wholesale.

Both behaviours are wanted, and which one is correct depends on the site. Sometimes the child means "forget what was inherited, this is the list". Sometimes it means "whatever was inherited, plus this" — and today it must restate the parent's entire list to say so, which duplicates the elements and silently goes stale when the parent changes.

This is not a case of picking the better default. PR #56 made arrays merge index-wise for every site, and #58 reverted it because it changed the meaning of configurations that relied on replacement. Both behaviours survived that revert; what is missing is a way for the author to choose between them **at the site**, rather than for the whole language at once.

Resolves #91.

## What Changes

This change introduces the **marker**: one of the reserved strings `$super` or `$super*`, written as a direct element of an array, that tells the inheritance stage how this array composes with the array it inherits.

- **`$super`** splices the inherited array's elements at the marker's position. Position expresses intent: `["$super", x]` appends, `[x, "$super"]` prepends, `[x, "$super", y]` wraps.
- **`$super*`** pairs the elements following it with the inherited elements index-wise and merges each pair. This recovers the behaviour PR #56 shipped, now confined to sites that ask for it.

Supporting rules:

- **The default is unchanged.** An array containing no marker replaces the inherited array exactly as it does today, which is the property #58 established as a requirement.
- **A marked array is a pending expression, not a value.** It carries unchanged through an ancestor that does not define the key, which is what lets an `$includes` fragment contribute to a document it knows nothing about. Cached file resolutions keep their markers verbatim.
- **Composition does not depend on how files are grouped.** The same ordered contributions render identically whether they are split across files or flattened into one, so extracting a mixin or inserting an intermediate layer cannot change the output. Composing a `$super*` delta with a marker on either side is an error, because that is precisely the case whose result would otherwise depend on the grouping.
- **A merge is defined only between values of the same kind.** Within a pair, object combines with object and atom with atom; a cross-kind pair is an error, so a misaligned pairing is reported rather than silently resolved. The requirement is on the pair, not on the keys inside a paired object merge.
- **A marker that never resolves is an error**, reported once composition completes, rather than being silently dropped.
- **The `$super` namespace is reserved** in array element position. A string beginning with `$super` followed by a non-identifier character must be a defined marker or it is an error, which keeps spellings such as `$super[1:]` and `$super?` available to gain meaning later. `$supervisor` is ordinary data, and `raw:$super` escapes a literal.

### Compatibility

No configuration changes meaning through composition: an unmarked array behaves exactly as before. There is one narrow break — **a document that contains the literal string `$super` as an array element**, which is data today and a marker afterwards. The remedy is `raw:$super`. This warrants a minor version bump and a changelog entry rather than being described as a change with no effect on existing documents.

### Deliberately not included

Each of the following is left in the error space, so that any of them can be added later as a compatible extension. The reasoning is recorded in `design.md`.

- Slicing (`$super[1:]`), which would make "override an element and also append" expressible.
- Key-based pairing (`$super*(name)`), which would match elements by a key field instead of by index.
- Markers in nested positions, under a path-based interpretation of what a marker refers to.
- Binding the inherited value so that `eval:` expressions can compute with it.
- A `merge` builtin exposed to JSON++ authors.

### Out of scope: marker-family strings outside array element position

A marker-family string written as an object key (`{"$super": 1}`), as a scalar value (`{"m": "$super"}`), or in an `$extends` or `$includes` filename slot is not addressed here. `$super` denotes a stream of elements rather than a value, so a marker in a value slot is wrong for the same reason `{"m": "$extends"}` is wrong — one rule covering every reserved word, which belongs with #92 rather than being reinvented per feature. Until #92 is resolved, such a string is emitted as data.

This does not weaken the cases that are in scope. A marker in an array nested inside an unmarked array, or inside a `$super*` queue element, is still an error here — not because of where it sits, but because nothing ever binds it, which the unresolved-marker rule already reports.

## Capabilities

### New Capabilities

`openspec/specs/` has no entries yet, so this change introduces the first one. It is scoped to array composition rather than to inheritance as a whole, leaving sibling capabilities such as object merging and file resolution to be specified by the changes that address them.

- `inheritance/array-composition`: how an array in a child document composes with the array it inherits — replacement by default, and the markers that select splicing or index-wise merging instead.

### Modified Capabilities

None. No capability is currently specified.

## Impact

- **`internal/json.go`** — `MergeObjects` currently recurses only when both sides are objects and falls through to child-replaces for every other pair, which is where arrays are replaced today. The existing `MergePolicy` parameter is the seam. The reverted implementation in commit `80908b2` is recoverable source material for the pairing engine.
- **Node pool** — cached resolutions must keep markers verbatim, and splicing must build fresh slices rather than share backing arrays.
- **`tools/etc/autotest/`** — new cases under `inheritance/` for the composing behaviours and under `negative/` for the error cases. This suite is the objective check, and every scenario in the spec delta should be traceable to a case here.
- **Documentation** — `tools/etc/docs/concepts/terminology.adoc` gains the terms this change introduces, and `evaluation-model.adoc` has a merge-semantics table and a future-consideration note from #73 / #75 that this change makes out of date.
- **Related issues** — #74 and #84 hold the earlier design discussion, treated here as source material rather than as specification. #56 / #58 are the reverted attempt. #76 / #77 recorded the same-kind merge principle. #63 / #73 / #75 are the documentation groundwork. #92 takes the out-of-scope positions above.
