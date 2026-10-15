# Changelog

This file records every notable change and feature in qfiller.

Format: newest entries at the top. Group each entry under **Added**,
**Changed**, **Fixed**, or **Removed**.

## [Unreleased]

### 2026-10-15: README moved to the project root

#### Changed
- `docs/README.md` moved to `README.md` at the project root, so it shows on the repository front page. Its link to the changelog now points to `docs/CHANGELOG.md`.

### 2026-10-15: Generated functions join with strings.Builder

#### Changed
- Generated functions build the result with a `strings.Builder` sized up front with `Grow`, instead of joining with `+`.
  - The Grow size is the literal byte count (an integer worked out at generation time) plus `len` of each parameter, multiplied by its number of uses, e.g. `b.Grow(62 + 2*len(schema))`.
  - Measured on `QUsers` (i7-10750H, Go 1.26.5): ~51 ns/op, down from ~63 ns/op with `+`. Both use 80 B and 1 alloc per op. A `Builder` without `Grow` was ~138 ns/op with 4 allocs, so `Grow` is required.
  - Templates with zero or one part don't use a builder: an empty template returns `""`, a template with no placeholders returns its constant (no allocation), and `{v}` returns `v`.
  - The output file imports `strings` only when some function uses a builder.
- Name clashes introduced by the builder code are handled:
  - The builder variable `b` gets `_` appended if a parameter or top-level name already uses it.
  - The `strings` import gets an alias (`strings_`, ...) if a parameter or top-level name is `strings`.
  - A `{len}` placeholder becomes the parameter `len_` (or longer, if taken) so the builtin `len` stays usable. Arguments are passed by position, so callers are not affected.
- Tests: golden output updated; the end-to-end test now covers `{b}`, `{strings}`, `{len}`, `{len_}` and a function named `strings`.
- README: generation rules describe the builder, and "A `strings.Builder` output mode" is removed from the next steps.

### 2026-10-15: `-src` flag wired into the CLI

#### Fixed
- `main.go` now has the `-src` flag described in the "Templates from source comments" entry. Before this, the entry listed it but the CLI didn't have it.
  - With `-src`, `-template` and `-func` are ignored. `-pkg` defaults to the scanned package and can still be set to override it.
  - Errors if no `//qfiller:` directives are found, instead of writing an empty file.
  - `-template` is now required only when `-src` is not set.

### 2026-10-15: Benchmark against strings.ReplaceAll

#### Added
- `bench` package: compares the generated `QUsers` (from the `{schema}` users/store join template) with `strings.ReplaceAll` on the same template.
  - The template is a `//qfiller:QUsers` directive in `bench.go`. `query_gen.go` is produced from it by `go generate ./bench`, which runs `qfiller -src .`.
  - The tests read the same directive back with `generator.Scan`, so `ReplaceAll` always runs on the exact template `QUsers` was generated from.
  - `TestSameResult` checks that both return the same string.
  - `BenchmarkReplaceAll` and `BenchmarkGenerated` run with `go test -bench . ./bench`.
  - First run (i7-10750H, Go 1.26.5): generated ~63 ns/op, ReplaceAll ~128 ns/op. Both use 80 B and 1 alloc per op.

### 2026-10-15: Templates from source comments

#### Added
- `generator.Scan`: parses Go files with `go/parser` and collects templates from `//qfiller:FuncName` comment directives.
  - Line form: template text after the name, plus following line comments in the same group (joined with newlines, up to the next directive).
  - Block form: `/*qfiller:FuncName ... */`, template is the rest of the comment.
  - Accepts files or directories; skips `_test.go` and generated files, requires a single package.
  - Errors carry the source position: missing/invalid name, empty template (including gofmt moving a directive to the end of a doc comment), invalid template, duplicate function name.
- CLI flag `-src`: generate from directives in a file or directory. The output package defaults to the scanned package unless `-pkg` is set.
- Tests for scanning (`generator/scan_test.go`).

### 2026-10-15: Template generator implemented

#### Added
- `generator.Parse`: splits a template into ordered `Literal` and `Variadic` parts.
  - Placeholder syntax is `{name}`, where `name` must be a valid Go identifier and not a keyword.
  - `{{` and `}}` escape literal braces.
  - Adjacent literal text is merged into one part.
  - Errors on an unclosed `{`, a lone `}`, an empty name, an invalid identifier or a keyword.
- `generator.Generate`: builds a Go file with `go/ast` and formats it with `go/format`.
  - Literal parts become a `const` block named `<funcName>Lit<N>`.
  - Variadic parts become `string` parameters, in order of first appearance; a repeated placeholder becomes a single parameter.
  - The function returns the parts joined with `+`.
  - Several `Spec`s can go into one file. Duplicate or invalid function names and invalid package names are rejected.
  - Constant names never clash with parameter names or other top-level names (`_` is appended when needed).
  - Output starts with the `// Code generated by qfiller. DO NOT EDIT.` header.
- CLI in `main.go` with the flags `-template`, `-func` (default `Fill`), `-pkg` (default `main`) and `-o` (default stdout).
- Tests: parser cases, parser error cases, golden output, generator error cases, and an end-to-end test that compiles and runs the generated code.

#### Changed
- `main.go`: replaced the GoLand starter code with the CLI.

### 2026-10-15: Docs

#### Added
- `docs/` directory with the project overview ([README.md](../README.md), now at the project root) and this changelog.
