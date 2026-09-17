## 1. The splice marker

- [x] 1.1 Choose the diagnostic wording for every error condition in the spec delta and record it in `design.md`, so that later negative cases match one settled format rather than inventing per-case text; verify by the wording being present in `design.md` before any negative case is written
- [x] 1.2 Add marker recognition to the inheritance stage, applying the reserved-namespace rule to strings in direct array element position at an ordinary key; verify unit tests in `internal/` cover `$super` and `$super*` as markers, `$super[1:]` and `$super?` as unknown-marker errors, and `$supervisor` as ordinary data
- [x] 1.3 Implement `$super` splicing through the `MergePolicy` seam in `internal/json.go`, constructing fresh slices rather than appending into an inherited backing array; verify autotest cases for append, prepend, wrap, and an empty inherited array
- [x] 1.4 Reject an array containing more than one marker; verify negative autotest cases for two `$super` and for `$super` mixed with `$super*`
- [x] 1.5 Carry a marked array unchanged through an ancestor that does not define the key, and keep markers verbatim in cached file resolutions; verify autotest cases for the grandparent-resolves case and for an `$includes` fragment composing with the including document
- [x] 1.6 Splice a marker present in the inherited value through verbatim, so two splice deltas compose into a third; verify an autotest case asserting the composed value is still pending
- [x] 1.7 Add grounding: report an error for any marker composition never resolved; verify autotest cases for no-ancestor-answers, a document inheriting nothing, a marker inside an array nested in an unmarked array, and a marker inside a `$super*` queue element
- [x] 1.8 Reject a marked array whose inherited value at that key is present and is not an array; verify a negative autotest case
- [x] 1.9 Verify `raw:$super` yields the literal string inside an array, and that a string produced by an `eval:` expression is never classified as a marker; add an autotest case for each
- [x] 1.10 Confirm an unmarked array still replaces the inherited array; verify the pre-existing inheritance autotest cases pass unchanged and add a case pinning the nested-array-inside-a-replaced-array behaviour

## 2. The pairing marker

- [ ] 2.1 Implement `$super*` pairing — literal prefix before the marker, index-wise pairing after it, unpaired elements surviving on both sides — recovering the index-wise recursion and keep-extras logic from the reverted commit `80908b2` where it fits; verify one autotest case per behaviour
- [ ] 2.2 Implement the pair rule by kind: object with object deep-merges, array with array takes the queue element, atom with atom takes the queue element, and `{}` keeps the inherited element unchanged; verify one autotest case per rule
- [ ] 2.3 Reject a pair whose sides are of different kinds, counting null as an atom; verify negative autotest cases for object-with-atom and array-with-object
- [ ] 2.4 Confirm kind agreement is required of the pair and not of keys inside a paired object merge, so a key whose values differ in kind is an override; verify an autotest case
- [ ] 2.5 Reject composing a `$super*` delta with a marked array on either side; verify negative autotest cases for a splice delta over a pairing delta and for a pairing delta over a splice delta
- [ ] 2.6 Confirm composition does not depend on file grouping; verify autotest cases for the layered layout and the flat layout rendering identically, and that both match the spec delta's expected result
- [ ] 2.7 Run `make test` and `tools/bin/autotest`; verify both pass and that every scenario in the spec delta is traceable to at least one autotest case

## 3. Documentation and release

- [ ] 3.1 Add the terms this change introduces — marker, marked array, inherited array, delta, splice, prefix, queue, pairing, kind, grounding — to `tools/etc/docs/concepts/terminology.adoc`, each defined once; verify no synonym for any of them appears in the documentation added by this change
- [ ] 3.2 Update the merge-semantics table and the future-consideration note in `tools/etc/docs/concepts/evaluation-model.adoc` left by #73 / #75, which this change makes out of date; verify the generated documentation builds and the note no longer describes array composition as future work
- [ ] 3.3 Document both markers in the reference documentation, including the reserved namespace and the two divergences from jq's `*`; verify the examples shown match autotest cases
- [ ] 3.4 Document that override-then-append is not expressible, and that restating inherited elements as padding silently becomes an override when the ancestor grows; verify the boundary and its example appear in the shipped documentation rather than only in `design.md`
- [ ] 3.5 Record the compatibility break for a literal `$super` array element, with `raw:` as the remedy, in the changelog, and bump the minor version; verify the changelog entry and version bump are present
