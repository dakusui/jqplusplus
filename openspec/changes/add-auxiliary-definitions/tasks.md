## Delivery plan

An implementation **review slice** is a focused pull request that makes one phase tractable to review. A **delivery unit** is the smallest set of changes that can truthfully be released and documented as complete.

Sections 1 through 4 are review slices in one delivery unit. Create them as a GitHub stack, with each pull request based on the preceding slice, and keep the entire implementation stack out of `main` until all four sections satisfy the agreed specification. Then merge the stack in dependency order. The planning pull request and final archive pull request remain separate from this implementation stack.

## 1. Auxiliary state and structural composition

- [ ] 1.1 Characterize the current `NodeEntryValue` cache/copy behavior and the extraction lifecycle of `$local` with focused unit tests before changing the resolved-node state; verify the new tests pass against the unchanged implementation
- [ ] 1.2 Extend `NodeEntryValue` with an auxiliary object, extract and validate the root `$aux` control object when each source is loaded, and keep the ordinary object and auxiliary object independently deep-copied in the node pool; verify unit tests cover an absent namespace, an empty namespace, every non-object JSON kind, and a `raw:$aux` ordinary key that is not extracted
- [ ] 1.3 Compose auxiliary objects alongside ordinary objects on every file-level and node-level `$extends` and `$includes` path, using the same merge direction and value-composition rules; verify unit tests cover inherited defaults, child overrides, recursive object composition, inclusion precedence, and contributions from more than two sources
- [ ] 1.4 Ground array-composition markers in the completed auxiliary object as well as the ordinary object; verify unit tests cover a resolved auxiliary marker and an unresolved auxiliary marker diagnostic
- [ ] 1.5 Run `go test ./internal`, `make build`, and the existing inheritance autotests; verify all pass before beginning expression-evaluation changes

## 2. Auxiliary expression evaluation

- [ ] 2.1 Add characterization tests for ordinary key-side and value-side evaluation, including `raw:`, `ref`, `reftag`, `$cur`, `$curexpr`, cycle detection, and recursive passes; verify the tests pin the existing input root and path behavior
- [ ] 2.2 Refactor the key-side and value-side walkers so the mutation target, jq read context, and current-path prefix can be supplied separately without changing ordinary-object behavior; verify the characterization tests and all existing evaluation unit tests pass unchanged
- [ ] 2.3 Implement eager auxiliary preparation as key-side evaluation followed by value-side evaluation against a refreshed synthetic view of the composed ordinary and auxiliary objects; verify unit tests cover computed auxiliary keys, computed auxiliary values, reads of structurally composed ordinary data, recursive passes, and failure of an unused invalid expression
- [ ] 2.4 Bind the prepared auxiliary object as the jq variable `$aux` for both ordinary key-side and value-side expressions; verify unit tests cover a computed output key, a computed output value, nested auxiliary objects, and jq's ordinary result for a missing auxiliary property
- [ ] 2.5 Make auxiliary values addressable to `ref` under paths rooted at `$aux`, and prefix auxiliary `$cur` and `$curexpr` paths with the same segment; verify unit tests cover a successful lazy auxiliary reference, a multi-definition reference chain, an auxiliary-reference cycle diagnostic, and the exact current paths inside nested auxiliary objects
- [ ] 2.6 Preserve the separation between control state and output data throughout evaluation; verify tests cover `$aux` omission, coexistence with a literal `raw:$aux` output key, and a destination-relative `$local` template that observes the final composed auxiliary namespace only after materialization
- [ ] 2.7 Run `go test ./internal`, `make build`, and the full pre-existing autotest suite; verify the evaluation refactoring introduces no regression before adding new end-to-end coverage

## 3. End-to-end behavior and YAML compatibility

- [ ] 3.1 Add positive autotest cases for direct `$aux` use on both expression sides, computed auxiliary keys and values, inherited defaults, child overrides, recursive auxiliary composition, inclusion precedence, and ordinary-data reads; verify each positive scenario in the specification is traceable to a case
- [ ] 3.2 Add negative autotest cases for every non-object root `$aux`, an unused invalid auxiliary expression, and circular auxiliary references; verify diagnostics identify the `$aux` path or reference chain required by the specification
- [ ] 3.3 Add autotest cases for an empty namespace, missing-property jq semantics, `raw:$aux` coexistence, `$local` coexistence, and nested ordinary objects named `$aux`; verify only the root unescaped control key has auxiliary meaning
- [ ] 3.4 Remove the recursive underscore-stripping filter from `tools/bin/yq++` and pass `jq++` output directly to the YAML emitter; verify a focused command-level test covers root and nested `_`-prefixed keys and compares the ordinary data emitted by `jq++` and `yq++`
- [ ] 3.5 Run `make test`, `make build`, `tools/bin/autotest`, and the command-level `yq++` tests; verify all specification scenarios are covered and both front-ends preserve underscore-prefixed ordinary fields

## 4. Documentation, migration, and release record

- [ ] 4.1 Define *auxiliary namespace* and *auxiliary definition* at their first use in `tools/etc/docs/concepts/terminology.adoc`, and use those exact terms throughout the added documentation; verify the terminology audit finds no competing name for either concept
- [ ] 4.2 Document `$aux` syntax, composed scope, precedence, eager key-then-value preparation, jq-variable access, `ref` paths, `$cur` paths, root-only reservation, and `raw:$aux` escaping in the evaluation model and syntax/reference material; verify every normative rule in the specification has a corresponding documentation passage or example
- [ ] 4.3 Document the distinction between `$aux` and `$local`, including why local-node templates retain destination-relative evaluation; verify the guidance contains a coexistence example and does not describe either construct as private, protected, or confidential
- [ ] 4.4 Replace documented `_`-prefixed holder patterns with `$aux` and add migration guidance for existing `yq++` users, while retaining legitimate underscore-prefixed data examples; verify a repository search finds no documentation that promises automatic underscore stripping
- [ ] 4.5 Record both compatibility breaks in `CHANGELOG.md`: reserving an unescaped root `$aux` and preserving former underscore-prefixed holders in `yq++`; verify the entry gives `raw:$aux` and moving holders into `$aux` as the respective migration actions
- [ ] 4.6 Run `tools/bin/gendoc`, inspect the generated documentation, then run `make test`, `make build`, `tools/bin/autotest`, the command-level `yq++` tests, `openspec validate --all --strict`, and `git diff --check`; verify every command passes and no generated page loses the literal `JSON{plus}{plus}` or `jq{plus}{plus}` spelling
