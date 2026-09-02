## Context

See `proposal.md` — Why, for the motivation, and `specs/inheritance/array-composition/spec.md` for the required behaviour. The terms used below — marker, marked array, inherited array, delta, splice, prefix, queue, pairing, kind, grounding — are defined in the spec delta.

Three facts about the current implementation shape the approach.

- **Composition happens before evaluation.** The pipeline in `tools/etc/docs/concepts/evaluation-model.adoc` resolves inheritance in its own stage, then evaluates `eval:` and `raw:` prefixes over the result, and no stage revisits earlier work. By the time an expression can run, the inherited array has already been discarded by the merge that replaced it.
- **Array replacement is not a decision the code makes.** `MergeObjects` in `internal/json.go` recurses only when both sides are objects; every other pair falls through to child-replaces. Arrays are replaced because they are not objects, not because anything chose that for them. The function already carries a `MergePolicy` parameter with a single value, which is the seam this change uses.
- **The reverted implementation is recoverable.** PR #56 (commit `80908b2`, reverted in `59282d9`) implemented index-wise recursive merge for arrays, in `mergeValues` / `mergeArrays`. Its index-wise recursion and keep-extras logic are the skeleton for pairing. What it lacks is marker detection, prefix handling, the cross-kind error path, and grounding — and it applied everywhere rather than where asked, which is why it was reverted.

## Goals / Non-Goals

**Goals:**

- Put the composition choice at the site, in a form where reading the child tells you what happens.
- Keep composition independent of how contributions are divided across files, so that refactoring the file layout cannot change the output.
- Keep the space of unwritten spellings large, so that the deferred capabilities below can be added later without breaking anyone.

**Non-Goals:**

- Changing what an unmarked array does.
- Making arrays composable through the evaluation layer.
- Validating marker-family strings outside array element position. See the proposal's out-of-scope section; that belongs with #92.

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

This ordering is also why `eval:`-produced strings are exempt from marker classification: composition and grounding both finish before value-side evaluation begins. The check is on what the author wrote, not on what the document renders.

### Two markers, not one

`$super` and `$super*` are not one feature plus an embellishment. #53 asked for index-wise merging, #56 delivered it, and #58 reverted it for being global rather than for being unwanted. Both behaviours survived that revert and both need a spelling; providing only the splice would leave the #53 case unanswered.

### Composition must be associative

Composing is a binary operation, `base ⊕ child`, applied repeatedly when a document has several contributions. **Associativity** is the property that `(a ⊕ b) ⊕ c` equals `a ⊕ (b ⊕ c)` — that the bracketing does not affect the result. It matters here because the author does not choose the bracketing: it falls out of how content happens to be divided across files, and out of the node pool, which resolves each file once, standalone.

This is not visible in any single render. A given set of files has exactly one bracketing, so nobody ever sees two answers to compare. It becomes visible across an edit intended to change nothing.

Take a base and two mixins — one enables TLS on the first server, one adds a server:

```
base.json    { "servers": [{"host":"a"}, {"host":"b"}] }
X.json       { "servers": ["$super*", {"tls": true}] }
Y.json       { "servers": ["$super", {"host":"c"}] }
```

The application pulls in both:

```
app.json     { "$extends": ["base.json"], "$includes": ["X.json", "Y.json"] }

base ⊕ X     pair {"host":"a"} with {"tls":true}; {"host":"b"} unpaired
          =  [{"host":"a","tls":true}, {"host":"b"}]
     ⊕ Y     append
          =  [{"host":"a","tls":true}, {"host":"b"}, {"host":"c"}]        3 servers
```

Someone notices X and Y always travel together and bundles them. Same mixins, same order, nothing about the contributions touched:

```
policy.json  { "$includes": ["X.json", "Y.json"] }
app.json     { "$extends": ["base.json"], "$includes": ["policy.json"] }
```

But `policy.json` is now resolved on its own, so `X ⊕ Y` happens before `base` is in scope. Were that composition allowed to flatten:

```
X ⊕ Y        ["$super*", {"tls":true}]  under  ["$super", {"host":"c"}]
          =  ["$super*", {"tls":true}, {"host":"c"}]     <- {"host":"c"} is now in the queue

base ⊕ that  {"host":"a"} x {"tls":true}   ->  {"host":"a","tls":true}
             {"host":"b"} x {"host":"c"}   ->  {"host":"c"}      merged, not appended
          =  [{"host":"a","tls":true}, {"host":"c"}]             2 servers
```

Server `b` is gone. A file reorganization dropped a server, and nothing in the diff points at why.

So the property the documentation can promise is: **how a configuration is split across files is a matter of taste, not of meaning.** That is what lets someone extract a shared mixin without re-verifying downstream output.

It is also what makes the node-pool cache correct rather than merely fast. Caching a file's standalone resolution *is* a choice of bracketing, so without associativity the answer would depend on whether a file happened to be cached — a bug, not an inconvenience.

Because the property is invisible within one render, it cannot be checked by inspecting a single document. The two scenarios in the spec delta — the layered layout and the flat layout, asserted to render identically — are the only thing pinning it. Removing them silently stops the property from being tested.

### A `$super*` delta may not compose with a marker on either side

The two markers behave differently under composition, and the difference decides this.

`$super` is a *variable occurrence*: it denotes the inherited value and means the same thing wherever it lands. So composing two splice deltas always yields something expressible as another splice delta — `["$super","a"]` under `["$super","c"]` is just `["$super","a","c"]`, still pending. Nothing new is ever needed.

`$super*` is a *positional delimiter*: its meaning is where it sits relative to the queue. Splicing a delimiter into new surroundings changes the parse. With an inherited `["$super*", {"a":1}]` and a child `["$super", {"b":2}]`, flattening gives `["$super*", {"a":1}, {"b":2}]`, which grounds against `[{"x":0},{"y":0}]` as `[{"x":0,"a":1},{"y":0,"b":2}]` — the child's contribution absorbed into the queue and pair-merged. The intended composition appends it instead: `[{"x":0,"a":1},{"y":0},{"b":2}]`. The other direction fails earlier, since pairing needs concrete elements and a length, and an unresolved splice has neither.

Representing these faithfully needs nesting the flat array syntax cannot express, so v1 refuses. This is not a taste call: it excludes exactly the compositions that would break associativity, which is why the bundling refactor above fails loudly at the point of the edit instead of quietly dropping a server. Some subcases are provably safe — a pure-prepend child never touches the queue — and stay errors for rule simplicity, recoverable later from the error space.

### Nested markers are rejected by grounding, not by a position rule

An earlier draft carried a requirement that a marker is valid only as a direct array element, with scenarios rejecting object keys, scalar values, and nested positions. That requirement is gone. Marker-family strings outside array element position belong with #92, and the two nested cases fall out of rules this change already has:

```
{ "m": [["$super", 5]] }              outer array unmarked -> replaces wholesale
                                      inner marker never bound -> unresolved -> error

{ "m": ["$super*", ["$super", 5]] }   pair is array x array -> queue element taken whole
                                      inner marker never bound -> unresolved -> error
```

Note the second depends on array-with-array taking the queue element rather than recursing into it. The two are coupled: adopting recursion later would remove this rejection, which is the point — nested composition then becomes expressible rather than erroring.

The alternative considered — and it is a good one — is to define a marker by its structural position: `$super` stands for the elements of the inherited array found at the path of the array containing it. That is uniform, works at any depth, and makes the nested case expressible without any special rule:

```
ancestor   { "m": [ [1,2], [3,4] ] }
child      { "m": [ ["$super", 5], [9] ] }
result     { "m": [ [1,2,5],   [9] ] }
```

Deferred rather than rejected on its merits, because it collides with `$super*`. The two use different alignment models: a path says the child element at index 1 corresponds to the inherited element at index 1, while `$super*` occupies a slot and pairs the element at index 1 with the inherited element at index 0. Every index after the marker disagrees by one. Resolving that means either excluding `$super*`, or defining paths to be computed after marker slots are discounted — a second alignment rule to specify and test. Since nested markers error today, the interpretation stays available.

### Strictness is chosen wherever the lenient option would be unrecoverable

Unresolved markers, unknown spellings in the `$super` namespace, cross-kind pairs, and an inherited value that is not an array are all errors. In each case the lenient alternative — silently yielding nothing, or emitting the marker as data — is a decision that cannot be reversed later, because configurations would come to depend on it. The strict choice can always be relaxed; the lenient one cannot be tightened.

Erroring on an unresolved marker in particular surfaces three real mistakes that grounding to an empty splice would hide: a mistyped key, a forgotten `$extends`, and a parent skipped silently by the optional-file `?` marker.

Reserved by the same argument, not committed: `$super?` and `$super*?` for a possible lenient variant meaning "splice if inherited, otherwise nothing", matching the optional-file `?` of `$extends`. The explicit workaround today is for the parent to declare `key: []`.

### Naming

- **`$super`** names the inheritance axis, following Jsonnet. It is deliberately distinct from the `parent` and `parentof` builtins, which address the structural path axis; the two axes must not share a word.
- **`$super*`** borrows jq's `*`, the recursive object-merge operator. The analogy is faithful in direction as well as operation: `A * B` in jq lets B win, and in `["$super*", o1]` the queue sits to the right of the marker and wins. The array reads as the expression it means — `[new] + ($super * [o1, o2])`.
- Two honest divergences from jq, worth documenting when this lands: jq errors on `array * array`, and this design lifts object-`*` index-wise into that gap; and jq's `*` on two atoms is arithmetic, where this design takes child-wins.
- `$super?` was rejected as the pairing marker's name because `?` already means optional in `$extends`. It is now reserved for exactly that meaning.

### Implementation shape

- Splicing and pairing build fresh slices. `MergeObjects` stores references into its result, so mutating an inherited `[]any` in place would corrupt the node-pool cache for every other document inheriting it.
- The node pool caches resolved files with their markers verbatim. A cached fragment whose marker had been resolved or dropped at cache time would arrive at the including document with its intent already destroyed.
- Grounding runs on the top-level object only, after node-level inheritance and before key-side evaluation. Per-file cached resolutions keep their markers.
- The value walker (`internal/obj.go`, `walkAnyPath`) already descends into arrays, so `raw:` unescaping inside arrays needs no new code.

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

→ Mitigation: state the boundary in the shipped documentation rather than leaving it to be discovered, and describe the padding idiom as unsafe when the ancestor may grow. The durable fixes are slicing and key-based pairing, both below.

**Index-wise pairing is fragile under reordering.** Inserting or moving an inherited element silently changes which queue element pairs with which.

→ Mitigation: the cross-kind error catches misalignment whenever the kinds differ, which is the common case for structured elements. It cannot catch a reorder among same-kind elements. Key-based pairing is the real answer and is deferred.

**A literal `$super` array element changes meaning.** Documents that use that exact string as data are affected.

→ Mitigation: `raw:$super`, a minor version bump, and a changelog entry. Recorded in the proposal rather than described as a change with no compatibility effect.

**There is no escape hatch for a composition the markers cannot express.** Evaluation cannot reach the inherited value. Parking it under a second key that nobody overrides does make it reachable by `ref`, but that key then appears in the output: `$local` is removed before processing, but it holds node definitions for `$extends` and `$includes` by name, not values an expression can address, and there is no other way to hide a key.

→ Mitigation: describe the boundary honestly. Do not document `eval:` as a general fallback, because it is not one today.

**Marker-family strings outside array element position stay silent until #92.** `{"m": "$super"}` — the missing-brackets mistake — emits `$super` as data in the meantime.

→ Mitigation: the window is bounded by #92, which covers every reserved word under one rule rather than each feature bringing its own.

**Terminology.** This change introduces concepts the documentation has no names for. The terms are defined once in the spec delta and belong in `tools/etc/docs/concepts/terminology.adoc` when the change lands.

## Future Concerns

Recorded here because they were considered during this change and would otherwise be lost. Each is currently in the error space or otherwise unclaimed, so each remains addable without breaking anyone.

- **Slicing — `$super[1:]`, `$super[:-1]`.** Generalises the marker from an opaque token to an expression over the inherited array, with the bare `$super` as sugar for the whole of it. It makes override-then-append expressible and correct as the ancestor grows: `["A", "$super[1:]", "z"]`. Its own questions — whether several markers may appear in one array, what overlapping slices do, whether an out-of-range slice is empty or an error, how a slice composes when carried through an ancestor that omits the key — are why it is not in this change. Reserved: `$super` followed by `[` is an error today.
- **Key-based pairing — `$super*(name)`.** Match elements by a key field instead of by index. Answers both the reorder fragility above and, in part, override-then-append.
- **Path-based markers.** See the decision above.
- **Binding the inherited value for evaluation.** Would let an author write arbitrary jq against what the merge consumed, covering everything positional markers cannot — for example `{"$merge": "<expr>"}` evaluated at merge time. Requires the inheritance stage to preserve what it currently discards, and the evaluation layer to bind more than `$cur` and `$curexpr`.
- **A `merge` builtin exposed to JSON++ authors.** Independently useful over values that survive composition, and orthogonal to this change. If it lands, the same-kind rule has to govern it too, or the language ends up with two divergent definitions of merging.
- **A way to hide a key from the output.** Without it, the helper-key technique described above pollutes what the document produces.

The last three are their own pains and belong in their own issues rather than in this change.

## Migration Plan

An unmarked array composes exactly as it does today, so no configuration changes meaning through composition. The one exception is the literal `$super` array element described above, which is why this ships as a minor version bump with a changelog entry pointing at `raw:` rather than as a change with no compatibility effect.

## Open Questions

- The exact wording and structure of the diagnostics. The spec requires that each condition is reported; it does not fix the message text. Settling this during implementation does not change the specs, the approach, or the task breakdown — but the negative autotest cases match on message content, so the wording should be chosen once, early in the first implementation phase, rather than per case.
