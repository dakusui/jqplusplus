## Purpose

Provide composed, processing-only data for JSON++ expression evaluation without occupying or deleting fields in the generated configuration.

## ADDED Requirements

### Requirement: Root auxiliary namespace
The system SHALL recognize a root `$aux` object as the auxiliary namespace. An **auxiliary definition** is a named value directly or transitively contained in that object; auxiliary definitions SHALL be available during processing and the control `$aux` object SHALL be absent from the evaluation result.

#### Scenario: Auxiliary definitions do not enter the result
- **WHEN** a document declares `{"$aux":{"service":"reviews"},"name":"eval:string:$aux.service"}`
- **THEN** its result contains `{"name":"reviews"}` and no control `$aux` field

#### Scenario: Empty auxiliary namespace
- **WHEN** a document declares an empty root `$aux` object
- **THEN** evaluation succeeds and no `$aux` field appears in the result

### Requirement: Auxiliary namespace shape
The root `$aux` control value MUST be an object. The system SHALL reject any other kind with a diagnostic that identifies `$aux` and the received kind.

#### Scenario: Non-object auxiliary namespace
- **WHEN** a document declares a root `$aux` value that is an array, atom, or null
- **THEN** evaluation fails with a diagnostic identifying `$aux` and the received kind

### Requirement: Composed auxiliary scope
The system SHALL compose `$aux` contributions from the entry document, `$extends`, and `$includes` into one auxiliary namespace using the same precedence and value-composition rules that apply to ordinary object fields. Every expression in the resulting document SHALL observe that composed namespace rather than a source-file-specific namespace.

#### Scenario: Child auxiliary definition overrides parent
- **WHEN** a parent defines `$aux.port` as `8000`, a child extends it, and the child defines `$aux.port` as `9000`
- **THEN** expressions inherited from the parent and expressions written in the child both observe `$aux.port` as `9000`

#### Scenario: Parent supplies an auxiliary default
- **WHEN** a parent defines `$aux.host` and an extending child does not override it
- **THEN** expressions in the composed document can read the inherited `$aux.host`

#### Scenario: Auxiliary objects compose recursively
- **WHEN** a parent defines `$aux.database` with `host` and `port` fields and a child overrides only `$aux.database.port`
- **THEN** the composed auxiliary definition retains the parent host and uses the child port

#### Scenario: Inclusion precedence applies to auxiliary definitions
- **WHEN** an included fragment and the including document contribute the same auxiliary definition
- **THEN** the included contribution wins according to the ordinary `$includes` precedence rule

### Requirement: Auxiliary preparation order
The system SHALL fully process key-side expressions and then value-side expressions in the composed auxiliary namespace before processing key-side expressions in the ordinary result object. Auxiliary expressions SHALL be evaluated against the structurally composed ordinary object and the current auxiliary namespace.

#### Scenario: Computed auxiliary value supplies an output key
- **WHEN** an auxiliary value expression produces an array of key names and an ordinary key expression reads that value through `$aux`
- **THEN** the ordinary key expression observes the processed array and emits the corresponding keys

#### Scenario: Computed auxiliary key supplies an output value
- **WHEN** an auxiliary key expression creates a named auxiliary definition and an ordinary value expression reads that definition through `$aux`
- **THEN** the ordinary value expression observes the definition under its computed name

#### Scenario: Auxiliary expression reads composed ordinary data
- **WHEN** an auxiliary value expression reads an ordinary field established by structural composition
- **THEN** it observes the structurally composed ordinary value

#### Scenario: Unused invalid auxiliary expression
- **WHEN** an auxiliary definition contains an invalid expression that no ordinary field references
- **THEN** evaluation fails because the auxiliary namespace is prepared in full

### Requirement: Auxiliary variable
The system SHALL bind the fully prepared auxiliary namespace to the jq variable `$aux` during both key-side and value-side evaluation of the ordinary object. Access through `$aux` SHALL use ordinary jq object-access behavior.

#### Scenario: Auxiliary value used by value-side expression
- **WHEN** an ordinary value expression reads an existing definition as `$aux.name`
- **THEN** the expression receives that definition's processed value

#### Scenario: Auxiliary value used by key-side expression
- **WHEN** an ordinary key expression reads an array of strings from `$aux.names`
- **THEN** the expression can emit one ordinary key for each string

#### Scenario: Missing auxiliary definition follows jq semantics
- **WHEN** an expression reads a name absent from the auxiliary object
- **THEN** the lookup produces jq's ordinary missing-object-field result

### Requirement: Auxiliary references and cycles
During auxiliary value preparation, the system SHALL make the auxiliary namespace addressable from `ref` by a path beginning with `$aux`, and SHALL apply the existing reference-resolution and circular-reference diagnostics to those paths.

#### Scenario: Computed auxiliary definition references another
- **WHEN** one auxiliary value expression uses `ref(["$aux","base"])` to read another auxiliary definition
- **THEN** the referenced definition is resolved using the ordinary `ref` behavior

#### Scenario: Circular auxiliary references
- **WHEN** auxiliary definitions refer back to one another through `ref`
- **THEN** evaluation fails with a circular-reference diagnostic identifying the auxiliary reference chain

### Requirement: Auxiliary current path
While evaluating an expression stored in the auxiliary namespace, the system SHALL expose `$cur` and `$curexpr` as paths rooted at the control `$aux` segment.

#### Scenario: Current path inside auxiliary namespace
- **WHEN** an expression is stored at `$aux.service.port`
- **THEN** `$cur` identifies `["$aux","service","port"]` and `$curexpr` identifies the equivalent path expression

### Requirement: Literal aux output key
The system SHALL preserve the existing `raw:` escape so that `raw:$aux` produces an ordinary literal output key named `$aux`; that literal key SHALL remain distinct from the control auxiliary namespace and SHALL survive output omission.

#### Scenario: Escaped literal aux key
- **WHEN** a document contains an ordinary root key named `raw:$aux`
- **THEN** the result contains a literal `$aux` key with its declared value

#### Scenario: Control and literal aux coexist
- **WHEN** a document declares both the control `$aux` object and a `raw:$aux` ordinary key
- **THEN** expressions receive the control object through the `$aux` variable and the result contains only the escaped literal `$aux` key

### Requirement: Local nodes remain distinct
The system SHALL continue to interpret `$local` as local-node definitions for `$extends` and `$includes`; auxiliary-definition processing SHALL NOT eagerly evaluate or otherwise alter a local-node template in place.

#### Scenario: Local template and auxiliary definitions coexist
- **WHEN** a document declares a `$local` template containing destination-relative expressions and also declares `$aux`
- **THEN** the local template is evaluated only after use at its destination while expressions there can observe the composed auxiliary namespace

### Requirement: Underscore-prefixed field preservation
The system SHALL treat `_`-prefixed ordinary keys as data and SHALL preserve them at every object depth in both `jq++` JSON output and `yq++` YAML output.

#### Scenario: Root underscore-prefixed fields survive YAML output
- **WHEN** `yq++` evaluates a document containing legitimate root `_id` and `_note` fields
- **THEN** both fields and their values appear in the YAML output

#### Scenario: Nested underscore-prefixed fields survive YAML output
- **WHEN** `yq++` evaluates a document containing an `_id` field inside an object nested in an array
- **THEN** the nested `_id` field and its value appear in the YAML output

#### Scenario: JSON and YAML front-ends agree on ordinary fields
- **WHEN** equivalent JSON++ and YAML inputs contain `_`-prefixed ordinary fields
- **THEN** `jq++` and `yq++` preserve the same ordinary data apart from output serialization format
