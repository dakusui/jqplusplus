## Purpose

Defines how an array in a child document composes with the array it inherits through `$extends` or `$includes`: replacement by default, and the reserved markers that let the author select splicing or index-wise merging at an individual site instead.

## ADDED Requirements

### Requirement: Arrays replace by default

A **marker** is one of the exact strings `$super` or `$super*`, written as a direct element of an array value at an ordinary key. An array containing a marker is a **marked array**; one containing none is **unmarked**. The **inherited array** is the value at the same key in the already-resolved parent, for `$extends`, or in the object being included into, for `$includes`.

When a child document defines an unmarked array at a key that an ancestor also defines, the system SHALL use the child's array as the composed result and discard the inherited array.

#### Scenario: An unmarked array replaces the inherited array
- **GIVEN** an ancestor defining `{"m": ["a", "b", "c"]}`
- **WHEN** a child inheriting from it defines `{"m": ["x"]}`
- **THEN** the composed result is `{"m": ["x"]}`

#### Scenario: Arrays nested inside a replaced array are not composed
- **GIVEN** an ancestor defining `{"m": [[1, 2], [3, 4]]}`
- **WHEN** a child inheriting from it defines `{"m": [[9]]}`
- **THEN** the composed result is `{"m": [[9]]}`

### Requirement: `$super` splices the inherited array

To **splice** is to substitute the inherited array's elements, in their inherited order, at the position the marker occupies. The system SHALL splice at each `$super` marker, so that the marker's position within the child's array determines where the inherited elements appear.

#### Scenario: Marker first appends the child's elements
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** the composed result is `{"m": ["a", "b", "z"]}`

#### Scenario: Marker last prepends the child's elements
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["z", "$super"]}`
- **THEN** the composed result is `{"m": ["z", "a", "b"]}`

#### Scenario: Marker in the middle wraps the inherited elements
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["y", "$super", "z"]}`
- **THEN** the composed result is `{"m": ["y", "a", "b", "z"]}`

#### Scenario: An empty inherited array contributes no elements
- **GIVEN** an ancestor defining `{"m": []}`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** the composed result is `{"m": ["z"]}`

### Requirement: An array composes with at most one marker

The system SHALL report an error when an array contains more than one marker, whether the markers are the same or different.

#### Scenario: Two splice markers are rejected
- **WHEN** a document contains `{"m": ["$super", "x", "$super"]}`
- **THEN** evaluation fails, reporting more than one marker in the array

#### Scenario: Mixing the two markers is rejected
- **WHEN** a document contains `{"m": ["$super*", "x", "$super"]}`
- **THEN** evaluation fails, reporting more than one marker in the array

### Requirement: `$super*` pairs the queue with the inherited elements

In an array containing `$super*`, the elements before the marker are the **prefix** and the elements after it are the **queue**. **Pairing** combines queue element *i* with inherited element *i*.

The system SHALL emit the prefix literally, SHALL pair each queue element with the inherited element at the same offset, and SHALL carry elements left unpaired on either side into the composed result in their original order.

#### Scenario: Paired objects are merged
- **GIVEN** an ancestor defining `{"m": [{"a": 1, "b": 2}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {"b": 20}]}`
- **THEN** the composed result is `{"m": [{"a": 1, "b": 20}]}`

#### Scenario: An empty object keeps the inherited element unchanged
- **GIVEN** an ancestor defining `{"m": [{"a": 1}, {"b": 2}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {}, {"b": 20}]}`
- **THEN** the composed result is `{"m": [{"a": 1}, {"b": 20}]}`

#### Scenario: Inherited elements beyond the queue survive
- **GIVEN** an ancestor defining `{"m": ["a", "b", "c"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** the composed result is `{"m": ["A", "b", "c"]}`

#### Scenario: Queue elements beyond the inherited array survive
- **GIVEN** an ancestor defining `{"m": ["a"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A", "B"]}`
- **THEN** the composed result is `{"m": ["A", "B"]}`

#### Scenario: The prefix is literal and does not pair
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["p", "$super*", "A"]}`
- **THEN** the composed result is `{"m": ["p", "A", "b"]}`

### Requirement: A pair is combined according to its kind

A **kind** is one of object, array, or atom; strings, numbers, booleans, and null are all atoms. The system SHALL combine each pair by kind: object with object deep-merges, array with array takes the queue element, atom with atom takes the queue element.

#### Scenario: Paired atoms take the queue element
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** the composed result is `{"m": ["A", "b"]}`

#### Scenario: Paired arrays take the queue element
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", [9]]}`
- **THEN** the composed result is `{"m": [[9]]}`

### Requirement: A pair of different kinds is an error

The system SHALL report an error when pairing combines an inherited element with a queue element of a different kind. This surfaces a misaligned pairing, which index-wise pairing cannot otherwise detect, as an error rather than a merge into the wrong element.

#### Scenario: Object paired with atom is rejected
- **GIVEN** an ancestor defining `{"m": [{"a": 1}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** evaluation fails, reporting that the pair at that position mixes kinds

#### Scenario: Array paired with object is rejected
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {"a": 1}]}`
- **THEN** evaluation fails, reporting that the pair at that position mixes kinds

### Requirement: Kind agreement is required of the pair, not of the keys within it

Once a pair is object with object, the system SHALL merge it as an ordinary object merge, in which a key whose inherited and queue values differ in kind is an override taking the queue element's value.

#### Scenario: A key inside a paired object may change kind
- **GIVEN** an ancestor defining `{"m": [{"name": "a", "opts": {"x": 1}}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {"opts": "none"}]}`
- **THEN** the composed result is `{"m": [{"name": "a", "opts": "none"}]}`

### Requirement: A marked array carries while the inherited array is absent

A marked array is a **delta**: a pending expression over the inherited array rather than a value. The system SHALL carry a delta unchanged through an ancestor that does not define the key, and SHALL preserve markers verbatim in any cached resolution of a file, so that a fragment resolved on its own reaches the document that includes it with its markers intact.

#### Scenario: An ancestor that omits the key does not consume the marker
- **GIVEN** a grandparent defining `{"m": ["a"]}` and a parent inheriting from it that does not define `m`
- **WHEN** a child inheriting from the parent defines `{"m": ["$super", "z"]}`
- **THEN** the composed result is `{"m": ["a", "z"]}`

#### Scenario: An included fragment composes with the including document
- **GIVEN** a fragment defining `{"m": ["$super", "platform"]}` and no `$extends`
- **WHEN** a document defines `{"$includes": ["fragment"], "m": ["app"]}`
- **THEN** the composed result is `{"m": ["app", "platform"]}`

### Requirement: Two splice deltas compose into a third delta

At any single merge step the child's marker is the one being resolved; a marker within the inherited value is not interpreted at that step. The system SHALL splice such a marker through verbatim, so that composing two splice deltas yields a delta that is still pending.

#### Scenario: Composing two splice deltas leaves a delta
- **GIVEN** an inherited value of `["$super", "a"]`
- **WHEN** a child inheriting it defines `{"m": ["$super", "c"]}`
- **THEN** the composed value at that key is `["$super", "a", "c"]`, still pending

### Requirement: Composition does not depend on how files are grouped

The system SHALL produce the same composed result for the same ordered contributions regardless of how they are divided across files, so that extracting a fragment, inserting an intermediate layer, or splitting a file does not change the output.

#### Scenario: An intermediate layer does not change the result
- **GIVEN** `m1` defining `{"m": ["$super", "m1"]}`, and `m2` defining `{"$extends": ["m1"], "m": ["$super", "m2"]}`
- **WHEN** a document defines `{"$includes": ["m2"], "m": ["base"]}`
- **THEN** the composed result is `{"m": ["base", "m1", "m2"]}`

#### Scenario: The same contributions included flat give the same result
- **GIVEN** `m1` defining `{"m": ["$super", "m1"]}`, and `m2` defining `{"m": ["$super", "m2"]}`
- **WHEN** a document defines `{"$includes": ["m1", "m2"], "m": ["base"]}`
- **THEN** the composed result is `{"m": ["base", "m1", "m2"]}`

### Requirement: A `$super*` delta does not compose with a marker on either side

The system SHALL report an error when a merge would compose a `$super*` delta with a marked array on the other side. Such a composition has no faithful representation as a single flat marked array, so its result would otherwise depend on how the contributions happen to be divided across files.

#### Scenario: A splice delta over a pairing delta is rejected
- **GIVEN** an inherited value of `["$super*", {"a": 1}]`
- **WHEN** a child inheriting it defines `{"m": ["$super", {"b": 2}]}`
- **THEN** evaluation fails, reporting that a `$super*` delta cannot compose with a marker

#### Scenario: A pairing delta over a splice delta is rejected
- **GIVEN** an inherited value of `["$super", {"a": 1}]`
- **WHEN** a child inheriting it defines `{"m": ["$super*", {"b": 2}]}`
- **THEN** evaluation fails, reporting that a `$super*` delta cannot compose with a marker

### Requirement: A marker that never resolves is an error

**Grounding** is the final pass over the top-level object, after node-level inheritance and before key-side evaluation. The system SHALL report an error for any marker that composition never resolved. Absence of an inherited array during composition is not an error; the delta carries.

#### Scenario: A marker no ancestor answers is rejected
- **GIVEN** an ancestor that does not define `m`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** evaluation fails, reporting an unresolved marker

#### Scenario: A marker in a document that inherits nothing is rejected
- **WHEN** a document with no `$extends` or `$includes` contains `{"m": ["$super"]}`
- **THEN** evaluation fails, reporting an unresolved marker

#### Scenario: A marker inside an array nested in an unmarked array is rejected
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": [["$super", 5]]}`
- **THEN** evaluation fails, reporting an unresolved marker, because the unmarked outer array replaced the inherited array and nothing bound the inner marker

#### Scenario: A marker inside a queue element is rejected
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", ["$super", 5]]}`
- **THEN** evaluation fails, reporting an unresolved marker, because the pair took the queue element whole and nothing bound the inner marker

### Requirement: An inherited value that is not an array is an error

The system SHALL report an error when a marked array's inherited value at that key is present and is not an array.

#### Scenario: Inheriting an object where a marked array composes is rejected
- **GIVEN** an ancestor defining `{"m": {"a": 1}}`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** evaluation fails, reporting that the inherited value is not an array

### Requirement: The `$super` namespace is reserved

In a direct array element position at an ordinary key, the system SHALL reject a string that begins with `$super` followed by a character that cannot appear in an identifier, unless the string is exactly a defined marker. A string beginning with `$super` followed by an identifier character SHALL be ordinary data. This keeps undefined spellings available to gain meaning later without changing any working configuration.

#### Scenario: An undefined spelling in the namespace is rejected
- **WHEN** a document contains `{"m": ["$super[1:]"]}`
- **THEN** evaluation fails, reporting an unknown marker

#### Scenario: A lenient spelling is rejected
- **WHEN** a document contains `{"m": ["$super?"]}`
- **THEN** evaluation fails, reporting an unknown marker

#### Scenario: A string that merely starts with the marker text is data
- **WHEN** a document contains `{"m": ["$supervisor"]}`
- **THEN** the composed result contains the string `$supervisor` unchanged

### Requirement: `raw:` escapes a literal marker

The system SHALL pass a `raw:`-prefixed string through composition untouched, and SHALL unescape it during value-side evaluation, so that a literal marker string can be emitted as data.

#### Scenario: The raw prefix yields the literal text
- **WHEN** a document contains `{"m": ["raw:$super"]}`
- **THEN** the composed result contains the string `$super` as data

### Requirement: Strings produced by evaluation are not markers

Composition and grounding both precede value-side evaluation, so the system SHALL NOT classify a string produced by an `eval:` expression as a marker. This is a composition-time check, not a check on the output.

#### Scenario: An evaluated string that spells a marker is data
- **WHEN** a document contains an `eval:` expression at an array element that produces the string `$super`
- **THEN** the composed result contains the string `$super` as data
