# go-lines

Line-oriented text processing: the pure per-line primitives (`Uppercase`, `WithPrefix`, `Numbered`, `Contains`) and a streaming `Process(ctx, reader, transform)` that scans an `io.Reader`, applies a `Transform`, honors context cancellation, and reports `Stats`. Lifted from `gomatic/modern-go-application`'s `internal/text`.

- Package is named `lines`; the primary value type is `Line` (so usage reads `lines.Line`, no stutter). Generic and CLI-agnostic — lives in `gomatic`.
- Errors use the package sentinel `type Error string`; the only emitted value is `const ErrReadInput Error`, wrapped with `%w` so the cause stays matchable under `errors.Is`. No `errors.New`/`fmt.Errorf` except `%w` wraps.
- Dependency-free (testify for tests). Gate: gofumpt, vet, staticcheck, govulncheck, gocognit ≤ 7, 100% coverage.
- Shared config (`Makefile`, `.golangci.yaml`, `.editorconfig`, `.gitignore`, `.github/`) is owned and pushed by `nicerobot/tools.repository` — do not edit in-tree; per-repo divergence goes in a `Makefile.local`.
