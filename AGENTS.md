# AGENTS

## Documentation Notes

- In AsciiDoc documentation, prefer `JSON{plus}{plus}` and `jq{plus}{plus}` over literal `JSON++` and `jq++`.
- This also applies inside backticks and other inline markup in `.adoc` files, where literal `++` may render incorrectly or disappear in generated HTML.
- For filename extensions and concrete command/file examples where the literal text matters, keep the real spelling such as `.json++` or `jq++` if that is what users must type.

## Terminology Discipline

- Define a term at its first use, or link to its definition in `tools/etc/docs/concepts/terminology.adoc`. This applies to issue comments and design proposals, not only shipped documentation.
- Use one name per concept. Do not alternate synonyms (for example "token" / "marker" / "positional token") for the same thing.
- New concepts introduced by a design belong in `terminology.adoc` when the design lands.
