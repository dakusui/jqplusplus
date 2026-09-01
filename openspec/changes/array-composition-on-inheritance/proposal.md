## Why

When a JSON++ document inherits from another through `$extends` or `$includes`, an object at a given key is merged with the inherited object, but an array at a given key replaces the inherited array wholesale.

Both behaviours are wanted, and which one is correct depends on the site. Sometimes the child means "forget what was inherited, this is the list". Sometimes it means "whatever was inherited, plus this" — and today it must restate the parent's entire list to say so, which duplicates the elements and silently goes stale when the parent changes.

This is not a case of picking the better default. PR #56 made arrays merge index-wise for every site, and #58 reverted it because it changed the meaning of configurations that relied on replacement. Both behaviours survived that revert; what is missing is a way for the author to choose between them **at the site**, rather than for the whole language at once.

Resolves #91.

## What Changes

This change introduces the **marker**: a reserved string, written as a direct element of an array, that tells the inheritance stage how this array composes with the array it inherits. Two markers are defined.

- **`$super`** — splices the inherited array's elements into this array at the marker's position. Position expresses intent: `["$super", x]` appends, `[x, "$super"]` prepends, `[x, "$super", y]` wraps.
- **`$super*`** — pairs the elements following it with the inherited array's elements index-wise, merging each pair. This recovers the behaviour PR #56 shipped, now confined to sites that ask for it.

Supporting rules:

- **The default is unchanged.** An array containing no marker replaces the inherited array exactly as it does today. This change is **not breaking**: every existing configuration keeps its current meaning, which is the property #58 was protecting.
- **Markers are valid only as direct elements of an array value.** A marker anywhere else — as an object key, as a scalar value, nested inside an element, or inside a value produced by `$super*` pairing — is an error.
- **A merge is defined only between values of the same kind.** Within a pair produced by `$super*`, object pairs with object and atom pairs with atom; a cross-kind pair is an error, so that a misaligned pairing is reported rather than silently resolved.
- **The `$super` namespace is reserved.** A string beginning with `$super` followed by a non-identifier character must match a defined marker exactly, or it is an error. This keeps spellings such as `$super[1:]` and `$super?` available to gain meaning later without breaking anyone. `$supervisor` is ordinary data, and `raw:$super` escapes a literal.
- **An unresolved marker is an error.** A marker whose inherited counterpart never materialises is reported at the end of composition rather than being silently dropped.

### Deliberately not included

Each of the following is left in the error space, so that any of them can be added later as a compatible extension. The reasoning is recorded in `design.md`.

- Slicing (`$super[1:]`), which would make "override an element and also append" expressible.
- Markers in nested positions, under a path-based interpretation of what a marker refers to.
- Binding the inherited value so that `eval:` expressions can compute with it.
- A `merge` builtin exposed to JSON++ authors.

**Out of scope: reserved keywords written where they have no meaning.** A marker in a meaningless position is an error under this change because the marker's own grammar has to say where it is valid. The existing behaviour of `$extends`, `$includes`, and `$local` in meaningless positions is untouched here; that hazard is #92 and is to be resolved on its own terms.

## Capabilities

### New Capabilities

`openspec/specs/` has no entries yet, so this change introduces the first one. It is scoped to array composition rather than to inheritance as a whole, leaving sibling capabilities such as object merging and file resolution to be specified by the changes that address them.

- `inheritance/array-composition`: how an array in a child document composes with the array it inherits — replacement by default, and the markers that select splicing or index-wise merging instead.

### Modified Capabilities

None. No capability is currently specified.

## Impact

- **`internal/json.go`** — `MergeObjects` currently recurses only when both sides are objects and falls through to child-replaces for every other pair, which is where arrays are replaced today. The existing `MergePolicy` parameter is the seam. The reverted implementation in commit `80908b2` is recoverable source material for the pairing engine.
- **Node aliasing** — splicing must build fresh slices rather than share backing arrays, because of the node-pool cache.
- **`tools/etc/autotest/`** — new cases under `inheritance/` for the composing behaviours and under `negative/` for the error cases. This suite is the objective check, and every scenario in the spec delta should be traceable to a case here.
- **Documentation** — `tools/etc/docs/concepts/terminology.adoc` gains the terms this change introduces, and the reference pages gain the markers.
- **Related issues** — #74 and #84 hold the earlier design discussion, treated here as source material. #56 / #58 are the reverted attempt. #76 / #77 recorded the same-kind merge principle. #63 / #73 / #75 are the documentation groundwork.
