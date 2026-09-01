## Context

See `proposal.md` — Why, for the motivation, and `specs/inheritance/array-composition/spec.md` for the required behaviour.

Three facts about the current implementation shape the approach.

- **Composition happens before evaluation.** The pipeline described in `tools/etc/docs/concepts/evaluation-model.adoc` resolves inheritance in its own stage, then evaluates `eval:` and `raw:` prefixes over the result, and no stage revisits earlier work. By the time an expression can run, the inherited array has already been discarded by the merge that replaced it.
- **Array replacement is not a decision the code makes.** `MergeObjects` in `internal/json.go` recurses only when both sides are objects; every other pair falls through to child-replaces. Arrays are replaced because they are not objects, not because anything chose that for them. The function already carries a `MergePolicy` parameter with a single value, which is the seam this change uses.
- **The reverted implementation is recoverable.** PR #56 (commit `80908b2`, reverted in `59282d9`) implemented index-wise recursive merge for arrays. Its pairing engine is source material for `$super*`; what has to change is that it applied everywhere rather than where asked.

## Goals / Non-Goals

**Goals:**

- Put the composition choice at the site, in a form where reading the child tells you what happens.
- Leave every existing configuration meaning exactly what it means today.
- Keep the space of unwritten spellings large, so that the deferred capabilities below can be added later without breaking anyone.

**Non-Goals:**

- Changing what an array without a marker does.
- Making arrays composable through the evaluation layer.
- Repairing the behaviour of `$extends`, `$includes`, and `$local` in positions where they have no meaning (#92).

## Decisions

### The marker is written by the child, not declared by the parent

The choice could instead live on the parent: a key declared as accumulating, with children then writing plain arrays that append.

Rejected. That is PR #56 with a smaller radius, and it has the property that made #56 harmful — an unchanged child silently changes meaning, and the meaning lives in a file the author of the child may not have open. Shrinking the radius from global to per-key does not remove the non-locality, it only makes it harder to find. A child-side marker is legible exactly where it is exercised.

### The marker sits in array-element position, not key position

A key-side spelling — `{"list+": [3]}` — would be more consistent with the existing reserved vocabulary, which lives entirely on object keys (`$extends`, `$includes`, `$local`), and it would make a misplaced keyword a key rather than a value.

Rejected, because key-side syntax cannot address every array in a document. An array nested directly inside another array has no key, so there is no key-side spelling that reaches it. Element position is total over array positions; key position covers a proper subset. Element position also lets the marker's index express append, prepend, and wrap, which a key-side form would have to encode separately.

### Composition is resolved in the inheritance stage

An alternative is to express composition with the existing evaluation machinery — a string element such as `eval:array:ref(...)`, resolved after inheritance.

Rejected for this purpose. Evaluation runs over the composed document, so it can reach any value that survived the merge but never the value the merge consumed. That is precisely the value this change is about. Evaluation remains the right tool for composing values that do survive, which is a different need.

### Two markers, not one

`$super` and `$super*` are not one feature plus an embellishment. #53 asked for index-wise merging, #56 delivered it, and #58 reverted it for being global rather than for being unwanted. Both behaviours survived that revert and both need a spelling; providing only the splice would leave the #53 case unanswered.

### A marker is valid only as a direct array element

The alternative considered — and it is a good one — is to define a marker by its structural position: `$super` stands for the elements of the inherited array found at the path of the array that contains it. That is uniform, works at any depth, and makes the nested case expressible without any special rule:

```
ancestor   { "m": [ [1,2], [3,4] ] }
child      { "m": [ ["$super", 5], [9] ] }
result     { "m": [ [1,2,5],   [9] ] }
```

Deferred rather than rejected on its merits, for two reasons.

First, it collides with `$super*`. The two use different alignment models: a path says the child element at index 1 corresponds to the inherited element at index 1, while `$super*` occupies a slot and pairs the element at index 1 with the inherited element at index 0. Every index after the marker disagrees by one. Resolving that means either excluding `$super*`, or defining paths to be computed after marker slots are discounted — a second alignment rule to specify and test.

Second, the restriction is in the error space. Markers in nested positions are rejected today, so the path interpretation can be adopted later as a compatible extension once the alignment question has been settled on its own.

### Strictness is chosen wherever the lenient option would be unrecoverable

Unresolved markers, unknown spellings in the `$super` namespace, cross-kind pairs, and an inherited value that is not an array are all errors. In each case the lenient alternative — silently yielding nothing, or emitting the marker as data — is a decision that cannot be reversed later, because configurations would come to depend on it. The strict choice can always be relaxed; the lenient one cannot be tightened. This is the same reasoning that keeps `$super[1:]` available below.

### Splicing builds fresh slices

Nodes are cached in a pool, so a spliced result that shares a backing array with an inherited value can alias it. Splice and pair operations construct new slices rather than appending into an existing one.

## Risks / Trade-offs

**Override-then-append is inexpressible, and the workaround for it fails silently.** `$super*` can override the element at a position; `$super` can append; they cannot co-occur, and the child cannot name the end of an array whose length it does not know. The workaround an author will find is to restate the inherited elements as padding:

```
ancestor  ["a", "b", "c"]
child     ["$super*", "A", "b", "c", "z"]     -->  ["A", "b", "c", "z"]

ancestor gains an element:
ancestor  ["a", "b", "c", "d"]
child     unchanged                            -->  ["A", "b", "c", "z"]
                                                                    ^ "z" now overrides "d"
```

Both `"z"` and `"d"` are atoms, so no cross-kind check fires and nothing is reported. An intended append has become a silent override — the same staleness this change exists to remove, reintroduced by the feature meant to fix it.

→ Mitigation: state the boundary in the shipped documentation rather than leaving it to be discovered, and describe the padding idiom as unsafe when the ancestor may grow. The durable fix is slicing (below).

**There is no escape hatch for a composition the markers cannot express.** Evaluation cannot reach the inherited value, for the reason given above. Parking the value under a second key that nobody overrides does make it reachable by `ref`, but that key then appears in the output: `$local` is removed before processing, but it holds node definitions for `$extends` and `$includes` by name, not values an expression can address, and there is no other way to hide a key.

→ Mitigation: describe the boundary honestly. Do not document `eval:` as a general fallback, because it is not one today.

**#92 may be read as partly addressed here.** A marker in a meaningless position is an error under this change, which resembles the hazard #92 describes.

→ Mitigation: the two are distinct and the proposal says so. A new construct must define where it is valid; that is its grammar. The behaviour of `$extends`, `$includes`, and `$local` in meaningless positions is unchanged by this work.

**Terminology.** This change introduces a concept the documentation has no name for. One name is used throughout — *marker* — and it belongs in `tools/etc/docs/concepts/terminology.adoc` when the change lands.

## Future Concerns

Recorded here because they were considered during this change and would otherwise be lost. Each is currently in the error space or otherwise unclaimed, so each remains addable without breaking anyone.

- **Slicing — `$super[1:]`, `$super[:-1]`.** Generalises the marker from an opaque token to an expression over the inherited array, with the bare `$super` as sugar for the whole of it. It makes override-then-append expressible and correct as the ancestor grows: `["A", "$super[1:]", "z"]`. Its own questions — whether several markers may appear in one array, what overlapping slices do, whether an out-of-range slice is empty or an error, how a slice composes when carried through an ancestor that omits the key — are why it is not in this change. Reserved: `$super` followed by `[` is an error today.
- **Path-based markers.** See the decision above.
- **Binding the inherited value for evaluation.** Would let an author write arbitrary jq against what the merge consumed, covering everything positional markers cannot. Requires the inheritance stage to preserve what it currently discards, and the evaluation layer to bind more than `$cur` and `$curexpr`.
- **A `merge` builtin exposed to JSON++ authors.** Independently useful over values that survive composition, and orthogonal to this change. If it lands, the same-kind rule has to govern it too, or the language ends up with two divergent definitions of merging.
- **A way to hide a key from the output.** Without it, the helper-key technique described above pollutes what the document produces.

The last three are their own pains and belong in their own issues rather than in this change.

## Migration Plan

None. An array containing no marker composes exactly as it does today, and every marker spelling is currently either an error or absent from real configurations. There is no configuration whose meaning changes, which is the property #58 established as a requirement.

## Open Questions

- The exact wording and structure of the diagnostics. The spec requires that each condition is reported; it does not fix the message text. Settling this during implementation does not change the specs, the approach, or the task breakdown — but the negative autotest cases match on message content, so the wording should be chosen once, early in the first implementation phase, rather than per case.
