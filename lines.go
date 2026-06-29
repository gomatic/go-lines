// Package lines provides line-oriented text processing primitives and a
// buffered line processor that applies a transform to each line of an input.
//
// The per-line operations are small and pure; Process owns the scanning,
// context-cancellation, counting, and joining. The package holds no CLI or
// orchestration logic and is reusable by any domain that needs line processing.
package lines

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type (
	// Line is a single line of input or output.
	Line string
	// Prefix is text prepended to a line.
	Prefix string
	// Filter is a substring a line must contain to be kept.
	Filter string
	// LineNumber is a one-based line position.
	LineNumber int
	// Output is the joined result of processing.
	Output string
	// Transform maps a numbered line to its processed form and whether to keep it.
	Transform func(Line, LineNumber) (Line, bool)
	// LineBytes is a size, in bytes, of a single scanned line.
	LineBytes int
)

// MaxLine is the largest single line Process accepts. It raises bufio's default
// 64 KiB scan ceiling to 1 MiB; a line longer than this makes Process fail with
// ErrReadInput wrapping bufio.ErrTooLong.
const MaxLine LineBytes = 1 << 20

// initialLine is the starting scan-buffer capacity, grown on demand up to
// MaxLine. It avoids preallocating MaxLine for inputs of ordinary lines.
const initialLine LineBytes = 64 << 10

// Stats reports the line counts observed during processing.
type Stats struct {
	Total LineNumber
	Kept  LineNumber
}

// Uppercase returns the line converted to uppercase.
func Uppercase(line Line) Line {
	return Line(strings.ToUpper(string(line)))
}

// WithPrefix returns the line with prefix prepended.
func WithPrefix(line Line, prefix Prefix) Line {
	return Line(string(prefix) + string(line))
}

// Numbered returns the line prefixed with its right-aligned line number.
func Numbered(line Line, number LineNumber) Line {
	return Line(fmt.Sprintf("%4d | %s", int(number), string(line)))
}

// Contains reports whether the line contains the filter substring.
func Contains(line Line, filter Filter) bool {
	return strings.Contains(string(line), string(filter))
}

// Process scans reader line by line, applies transform, and joins the kept
// lines with "\n". It stops early if ctx is cancelled and reports the total and
// kept line counts.
//
// Process is a buffered line processor, not a streaming one: every kept line is
// retained in memory and joined into a single Output, so peak memory is
// O(input).
//
// A trailing newline is not round-tripped: lines are joined with "\n" and no
// terminator, so "a\nb\nc\n" and "a\nb\nc" both yield "a\nb\nc". CRLF is
// normalized to LF, because the scanner strips a trailing "\r" from each line.
//
// A single line longer than MaxLine fails with ErrReadInput wrapping
// bufio.ErrTooLong.
func Process(ctx context.Context, reader io.Reader, transform Transform) (Output, Stats, error) {
	lines, stats, err := scan(ctx, reader, transform)
	if err != nil {
		return "", Stats{}, err
	}
	return Output(strings.Join(lines, "\n")), stats, nil
}

// scan reads each line, applies transform, and collects the kept results.
func scan(ctx context.Context, reader io.Reader, transform Transform) ([]string, Stats, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, int(initialLine)), int(MaxLine))
	kept := []string{}
	var stats Stats
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return nil, Stats{}, err
		}
		stats.Total++
		if line, ok := transform(Line(scanner.Text()), stats.Total); ok {
			kept = append(kept, string(line))
			stats.Kept++
		}
	}
	return kept, stats, scanError(scanner.Err())
}

// scanError wraps a scanner failure in the package sentinel.
func scanError(err error) error {
	if err != nil {
		return fmt.Errorf("%w: %w", ErrReadInput, err)
	}
	return nil
}
