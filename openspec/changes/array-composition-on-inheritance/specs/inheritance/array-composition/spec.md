## Purpose

Defines how an array in a child document composes with the array it inherits through `$extends` or `$includes`: replacement by default, and the reserved markers that let the author select splicing or index-wise merging at an individual site instead.

## ADDED Requirements

### Requirement: Arrays replace by default

When a child document defines an array at a key that an ancestor also defines, and that array contains no marker, the system SHALL use the child's array as the composed result and discard the inherited array.

A **marker** is one of the reserved strings defined by this capability, written as a direct element of an array, that selects how the array composes with the array it inherits.

#### Scenario: Child array with no marker replaces the inherited array
- **GIVEN** an ancestor defining `{"m": ["a", "b", "c"]}`
- **WHEN** a child inheriting from it defines `{"m": ["x"]}`
- **THEN** the composed result is `{"m": ["x"]}`

#### Scenario: Arrays nested inside a replaced array are not composed
- **GIVEN** an ancestor defining `{"m": [[1, 2], [3, 4]]}`
- **WHEN** a child inheriting from it defines `{"m": [[9]]}`
- **THEN** the composed result is `{"m": [[9]]}`

### Requirement: `$super` splices the inherited array

The system SHALL replace a `$super` marker with the elements of the inherited array, in their inherited order, at the position the marker occupies. The marker's position within the child's array SHALL determine where the inherited elements appear in the composed result.

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

### Requirement: `$super*` merges the child's elements with the inherited elements index-wise

The system SHALL treat the elements preceding a `$super*` marker as a literal prefix of the composed result, and SHALL pair each element following the marker with the inherited element at the same offset, merging each pair. Elements left unpaired on either side SHALL survive into the composed result in their original order.

#### Scenario: Paired objects are merged
- **GIVEN** an ancestor defining `{"m": [{"a": 1, "b": 2}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {"b": 20}]}`
- **THEN** the composed result is `{"m": [{"a": 1, "b": 20}]}`

#### Scenario: Paired atoms take the child's value
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** the composed result is `{"m": ["A", "b"]}`

#### Scenario: An empty object keeps the inherited element unchanged
- **GIVEN** an ancestor defining `{"m": [{"a": 1}, {"b": 2}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {}, {"b": 20}]}`
- **THEN** the composed result is `{"m": [{"a": 1}, {"b": 20}]}`

#### Scenario: Inherited elements beyond the child's elements survive
- **GIVEN** an ancestor defining `{"m": ["a", "b", "c"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** the composed result is `{"m": ["A", "b", "c"]}`

#### Scenario: Child elements beyond the inherited elements survive
- **GIVEN** an ancestor defining `{"m": ["a"]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A", "B"]}`
- **THEN** the composed result is `{"m": ["A", "B"]}`

#### Scenario: Elements before the marker are a literal prefix
- **GIVEN** an ancestor defining `{"m": ["a", "b"]}`
- **WHEN** a child inheriting from it defines `{"m": ["p", "$super*", "A"]}`
- **THEN** the composed result is `{"m": ["p", "A", "b"]}`

#### Scenario: Paired arrays take the child's value
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", [9]]}`
- **THEN** the composed result is `{"m": [[9]]}`

### Requirement: A pair of different kinds is an error

The system SHALL report an error when `$super*` pairs an inherited element with a child element of a different kind, where the kinds are object, array, and atom. A null value SHALL count as an atom.

#### Scenario: Object paired with atom is rejected
- **GIVEN** an ancestor defining `{"m": [{"a": 1}]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", "A"]}`
- **THEN** evaluation fails, reporting that the pair at that position mixes kinds

#### Scenario: Array paired with object is rejected
- **GIVEN** an ancestor defining `{"m": [[1, 2]]}`
- **WHEN** a child inheriting from it defines `{"m": ["$super*", {"a": 1}]}`
- **THEN** evaluation fails, reporting that the pair at that position mixes kinds

### Requirement: A marker is valid only as a direct element of an array

The system SHALL accept a marker only as a direct element of an array value, and SHALL report an error for a marker in any other position, including as an object key, as a scalar value, as an element of an array nested inside another array, and anywhere within a value that `$super*` pairs.

#### Scenario: Marker as an object key is rejected
- **WHEN** a document contains `{"$super": 1}`
- **THEN** evaluation fails, reporting the marker in an invalid position

#### Scenario: Marker as a scalar value is rejected
- **WHEN** a document contains `{"m": "$super"}`
- **THEN** evaluation fails, reporting the marker in an invalid position

#### Scenario: Marker inside a nested array is rejected
- **WHEN** a document contains `{"m": [["$super", 5]]}`
- **THEN** evaluation fails, reporting the marker in an invalid position

#### Scenario: Marker inside a paired value is rejected
- **WHEN** a document contains `{"m": ["$super*", ["$super", 5]]}`
- **THEN** evaluation fails, reporting the marker in an invalid position

### Requirement: An array composes with at most one marker

The system SHALL report an error when an array contains more than one marker, whether the markers are the same or different.

#### Scenario: Two splice markers are rejected
- **WHEN** a document contains `{"m": ["$super", "x", "$super"]}`
- **THEN** evaluation fails, reporting more than one marker in the array

#### Scenario: Mixing the two markers is rejected
- **WHEN** a document contains `{"m": ["$super*", "x", "$super"]}`
- **THEN** evaluation fails, reporting more than one marker in the array

### Requirement: The `$super` namespace is reserved

The system SHALL reject a string in a direct array element position that begins with `$super` followed by a character that cannot appear in an identifier, unless the string matches a defined marker exactly. A string beginning with `$super` followed by an identifier character SHALL be ordinary data. The `raw:` prefix SHALL yield the literal text.

#### Scenario: An undefined marker spelling is rejected
- **WHEN** a document contains `{"m": ["$super[1:]"]}`
- **THEN** evaluation fails, reporting an unknown marker

#### Scenario: A string that merely starts with the marker text is data
- **WHEN** a document contains `{"m": ["$supervisor"]}`
- **THEN** the composed result contains the string `$supervisor` unchanged

#### Scenario: The raw prefix escapes a literal marker
- **WHEN** a document contains `{"m": ["raw:$super"]}`
- **THEN** the composed result contains the string `$super` as data

### Requirement: A marker with no inherited array is an error

The system SHALL carry an unresolved marker forward while composing an inheritance chain, so that an ancestor that does not define the key does not consume the marker. The system SHALL report an error for a marker that is still unresolved when composition of the document completes.

#### Scenario: An ancestor that omits the key does not resolve the marker
- **GIVEN** a grandparent defining `{"m": ["a"]}` and a parent inheriting from it that does not define `m`
- **WHEN** a child inheriting from the parent defines `{"m": ["$super", "z"]}`
- **THEN** the composed result is `{"m": ["a", "z"]}`

#### Scenario: A marker no ancestor answers is rejected
- **GIVEN** an ancestor that does not define `m`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** evaluation fails, reporting an unresolved marker

#### Scenario: A marker in a document that inherits nothing is rejected
- **WHEN** a document with no `$extends` or `$includes` contains `{"m": ["$super"]}`
- **THEN** evaluation fails, reporting an unresolved marker

### Requirement: An inherited value of the wrong kind is an error

The system SHALL report an error when an array containing a marker inherits a value at that key that is not an array.

#### Scenario: Inheriting an object where an array is composed is rejected
- **GIVEN** an ancestor defining `{"m": {"a": 1}}`
- **WHEN** a child inheriting from it defines `{"m": ["$super", "z"]}`
- **THEN** evaluation fails, reporting that the inherited value is not an array
