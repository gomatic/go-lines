# go-lines

Line-oriented text processing: the pure per-line primitives (`Uppercase`, `WithPrefix`, `Numbered`, `Contains`) and a streaming `Process(ctx, reader, transform)` that scans an `io.Reader`, applies a `Transform`, honors context cancellation, and reports `Stats`. Lifted from `gomatic/modern-go-application`'s `internal/text`.

- Package is named `lines`; the primary value type is `Line` (so usage reads `lines.Line`, no stutter). Generic and CLI-agnostic — lives in `gomatic`.
- Errors use the ecosystem sentinel mechanism from [gomatic/go-error](https://github.com/gomatic/go-error): the only emitted value is `const ErrReadInput errs.Const`, and causes are wrapped via `ErrReadInput.With(cause)` so both the sentinel and the cause stay matchable under `errors.Is`. No `errors.New`/`fmt.Errorf` — ever.
- Dependencies: `gomatic/go-error` only (testify for tests). Gate: gofumpt, vet, staticcheck, govulncheck, gocognit ≤ 7, 100% coverage.
- Shared config (`Makefile`, `.golangci.yaml`, `.editorconfig`, `.gitignore`, `.github/`) is owned and pushed by `nicerobot/tools.repository` — do not edit in-tree; per-repo divergence goes in a `Makefile.local`.
