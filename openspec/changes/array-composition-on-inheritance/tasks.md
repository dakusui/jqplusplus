## 1. Marker recognition and the splice marker

- [ ] 1.1 Choose the diagnostic wording for every error condition in the spec delta and record it in `design.md`, so that later negative cases match one settled format rather than inventing per-case text; verify by the wording being present in `design.md` before any negative case is written
- [ ] 1.2 Add marker recognition to the inheritance stage, applying the reserved-namespace rule to strings in direct array element position; verify unit tests in `internal/` cover `$super` and `$super*` as markers, `$super[1:]` as an unknown-marker error, and `$supervisor` as ordinary data
- [ ] 1.3 Reject a marker as an object key, as a scalar value, and as an element of an array nested inside another array; verify one negative autotest case per position under `tools/etc/autotest/negative/`
- [ ] 1.4 Reject an array containing more than one marker; verify a negative autotest case
- [ ] 1.5 Verify `raw:$super` yields the literal string inside an array, adding an autotest case under `tools/etc/autotest/inheritance/`
- [ ] 1.6 Implement `$super` splicing through the `MergePolicy` seam in `internal/json.go`, constructing fresh slices rather than appending into an inherited backing array; verify autotest cases for append, prepend, wrap, and an empty inherited array
- [ ] 1.7 Carry an unresolved marker through an ancestor that does not define the key, and report an error when it is still unresolved once composition completes; verify autotest cases for the grandparent-resolves case, the no-ancestor-answers case, and a document that inherits nothing
- [ ] 1.8 Reject a marked array whose inherited value at that key is not an array; verify a negative autotest case
- [ ] 1.9 Confirm that an array with no marker still replaces the inherited array; verify the pre-existing inheritance autotest cases pass unchanged and add a case pinning the nested-array-inside-a-replaced-array behaviour

## 2. The index-wise merge marker

- [ ] 2.1 Implement `$super*` pairing — literal prefix before the marker, index-wise pairing after it, unpaired elements surviving on both sides — recovering the pairing engine from the reverted commit `80908b2` where it fits; verify autotest cases for each of those four behaviours
- [ ] 2.2 Implement the pair merge rules: object with object deep-merges, atom with atom takes the child's value, array with array takes the child's value, and `{}` keeps the inherited element unchanged; verify one autotest case per rule
- [ ] 2.3 Reject a pair whose two sides are of different kinds, counting null as an atom; verify negative autotest cases for object-with-atom and array-with-object
- [ ] 2.4 Reject a marker appearing anywhere inside a value that `$super*` pairs, and reject an array mixing the two markers; verify a negative autotest case for each
- [ ] 2.5 Confirm the whole capability holds together by running `make test` and `tools/bin/autotest`; verify both pass with every scenario in the spec delta traceable to at least one autotest case

## 3. Documentation

- [ ] 3.1 Add the term *marker* to `tools/etc/docs/concepts/terminology.adoc`, defined once and used consistently; verify no synonym for it appears elsewhere in the documentation added by this change
- [ ] 3.2 Document both markers in the reference documentation, including the position rule and the reserved namespace; verify the generated documentation builds and the examples shown match autotest cases
- [ ] 3.3 Document that override-then-append is not expressible, and that restating inherited elements as padding silently becomes an override when the ancestor grows; verify the boundary and its example appear in the shipped documentation rather than only in `design.md`
