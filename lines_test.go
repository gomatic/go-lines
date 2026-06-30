package lines

import (
	"bufio"
	"context"
	"errors"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
)

func TestUppercase(t *testing.T) {
	t.Parallel()
	assert.New(t).Equal(Line("HELLO WORLD"), Uppercase("hello world"))
}

func TestWithPrefix(t *testing.T) {
	t.Parallel()
	assert.New(t).Equal(Line(">> line"), WithPrefix("line", ">> "))
}

func TestNumbered(t *testing.T) {
	t.Parallel()
	assert.New(t).Equal(Line("   1 | line"), Numbered("line", 1))
}

func TestContains(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		line   Line
		filter Filter
		want   bool
	}{
		{name: "match", line: "keep this", filter: "keep", want: true},
		{name: "no match", line: "drop this", filter: "keep", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.New(t).Equal(tt.want, Contains(tt.line, tt.filter))
		})
	}
}

// keepAll is a transform that keeps every line unchanged.
func keepAll(line Line, _ LineNumber) (Line, bool) { return line, true }

func TestProcess(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		input     string
		transform Transform
		want      Output
		wantStats Stats
	}{
		{
			name:      "passthrough",
			input:     "a\nb\nc",
			transform: keepAll,
			want:      "a\nb\nc",
			wantStats: Stats{Total: 3, Kept: 3},
		},
		{
			name:      "empty input",
			input:     "",
			transform: keepAll,
			want:      "",
			wantStats: Stats{Total: 0, Kept: 0},
		},
		{
			// bufio.ScanLines strips a trailing \r, so CRLF input is
			// normalized to LF (lossy CRLF->LF) in the output.
			name:      "crlf normalized to lf",
			input:     "a\r\nb",
			transform: keepAll,
			want:      "a\nb",
			wantStats: Stats{Total: 2, Kept: 2},
		},
		{
			// A trailing newline is not round-tripped: lines are joined
			// with \n and no terminator, so this yields the same output as
			// the "passthrough" case above ("a\nb\nc").
			name:      "trailing newline is dropped",
			input:     "a\nb\nc\n",
			transform: keepAll,
			want:      "a\nb\nc",
			wantStats: Stats{Total: 3, Kept: 3},
		},
		{
			// Input that is only newline terminators yields one empty line
			// per terminator, joined back to terminators minus the dropped
			// trailing one: "\n\n\n" -> three empty lines -> "\n\n".
			name:      "only newlines yields empty lines",
			input:     "\n\n\n",
			transform: keepAll,
			want:      "\n\n",
			wantStats: Stats{Total: 3, Kept: 3},
		},
		{
			// Multibyte UTF-8 is passed through byte-for-byte; the scanner
			// splits on the LF byte only and never on bytes inside a rune.
			name:      "unicode multibyte passthrough",
			input:     "héllo\n世界\nنص",
			transform: keepAll,
			want:      "héllo\n世界\nنص",
			wantStats: Stats{Total: 3, Kept: 3},
		},
		{
			// Embedded NUL bytes are ordinary content: only LF delimits
			// lines, so NULs are preserved verbatim within a line.
			name:      "embedded NUL bytes preserved",
			input:     "a\x00b\nc\x00",
			transform: keepAll,
			want:      "a\x00b\nc\x00",
			wantStats: Stats{Total: 2, Kept: 2},
		},
		{
			// A lone CR not followed by LF is not a line terminator and is
			// preserved, unlike the trailing CR of a CRLF pair.
			name:      "lone CR is not a terminator",
			input:     "a\rb\nc",
			transform: keepAll,
			want:      "a\rb\nc",
			wantStats: Stats{Total: 2, Kept: 2},
		},
		{
			name:  "filters drop lines and renumber kept",
			input: "keep\ndrop\nkeep",
			transform: func(line Line, number LineNumber) (Line, bool) {
				if line != "keep" {
					return "", false
				}
				return Numbered(line, number), true
			},
			want:      "   1 | keep\n   3 | keep",
			wantStats: Stats{Total: 3, Kept: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			want := assert.New(t)

			output, stats, err := Process(context.Background(), strings.NewReader(tt.input), tt.transform)

			want.NoError(err)
			want.Equal(tt.want, output)
			want.Equal(tt.wantStats, stats)
		})
	}
}

func TestProcessReportsReadError(t *testing.T) {
	t.Parallel()
	const boom Error = "read exploded"

	output, stats, err := Process(context.Background(), iotest.ErrReader(boom), keepAll)

	want := assert.New(t)
	want.ErrorIs(err, ErrReadInput)
	want.ErrorIs(err, boom, "the underlying read error must be wrapped")
	want.Empty(output)
	want.Equal(Stats{}, stats)
}

// failAfterReader yields its data on the first Read, then fails every
// subsequent Read with err. It injects a mid-stream read failure: the scanner
// successfully tokenizes the buffered lines before the error surfaces.
type failAfterReader struct {
	err  error
	data []byte
	done bool
}

func (r *failAfterReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, r.err
	}
	r.done = true
	return copy(p, r.data), nil
}

// TestProcessReportsMidStreamReadError proves the discard-on-error contract:
// even when lines were read successfully before the reader fails, Process
// surfaces ErrReadInput wrapping the cause and returns no partial output or
// stats. This is stronger than the error-on-first-read case, which never
// accumulates any lines.
func TestProcessReportsMidStreamReadError(t *testing.T) {
	t.Parallel()
	const boom Error = "read exploded mid-stream"
	reader := &failAfterReader{data: []byte("a\nb\nc\n"), err: boom}

	output, stats, err := Process(context.Background(), reader, keepAll)

	want := assert.New(t)
	want.ErrorIs(err, ErrReadInput)
	want.ErrorIs(err, boom, "the underlying read error must be wrapped")
	want.Empty(output, "no partial output is returned on a mid-stream failure")
	want.Equal(Stats{}, stats, "stats are reset on a mid-stream failure")
}

// TestProcessAcceptsLargeLine proves the scan buffer is raised above bufio's
// default 64 KiB ceiling: a line larger than that default but within MaxLine is
// processed without error. Without the scanner.Buffer override this would fail
// with bufio.ErrTooLong, so it guards the raise against regression.
func TestProcessAcceptsLargeLine(t *testing.T) {
	t.Parallel()
	line := strings.Repeat("x", 128<<10) // 128 KiB > bufio's 64 KiB default

	output, stats, err := Process(context.Background(), strings.NewReader(line), keepAll)

	want := assert.New(t)
	want.NoError(err)
	want.Equal(Output(line), output)
	want.Equal(Stats{Total: 1, Kept: 1}, stats)
}

// TestProcessRejectsOverlongLine pins the documented MaxLine ceiling: a line
// exceeding it surfaces ErrReadInput wrapping bufio.ErrTooLong.
func TestProcessRejectsOverlongLine(t *testing.T) {
	t.Parallel()
	line := strings.Repeat("x", 2*int(MaxLine)) // unambiguously over the limit

	output, stats, err := Process(context.Background(), strings.NewReader(line), keepAll)

	want := assert.New(t)
	want.ErrorIs(err, ErrReadInput)
	want.ErrorIs(err, bufio.ErrTooLong, "the underlying scanner cause must be wrapped")
	want.Empty(output)
	want.Equal(Stats{}, stats)
}

func TestProcessStopsOnCancelledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	output, stats, err := Process(ctx, strings.NewReader("a\nb\nc"), keepAll)

	want := assert.New(t)
	want.ErrorIs(err, context.Canceled)
	want.Empty(output)
	want.Equal(Stats{}, stats)
}

// TestErrorString covers the sentinel type's Error method directly.
func TestErrorString(t *testing.T) {
	t.Parallel()
	assert.New(t).Equal("failed to read input", ErrReadInput.Error())
}

// FuzzProcess exercises the line scanner against arbitrary byte input under the
// identity transform and asserts the structural invariants that hold for every
// input. Idempotence is deliberately NOT asserted: a trailing empty line is
// unrepresentable once the joining terminator is dropped, so "\n\n\n" (three
// empty lines) becomes "\n\n" (two), and re-processing is not a fixed point.
// The invariants that DO hold for every input are:
//   - Process never panics.
//   - The only error a string reader can produce is the MaxLine overflow, which
//     is the documented ErrReadInput.
//   - The identity transform keeps exactly as many lines as it sees.
//   - Each scanned line is itself newline-free (the scanner splits on every LF),
//     and the lines are joined with a single LF, so the output holds exactly
//     one separator between lines: max(Total-1, 0) newlines in all.
func FuzzProcess(f *testing.F) {
	for _, seed := range []string{
		"",
		"\n",
		"\n\n\n",
		"a\nb\nc",
		"a\nb\nc\n",
		"a\r\nb\r\nc",
		"a\rb",
		"héllo\n世界\nنص",
		"a\x00b\nc\x00",
		strings.Repeat("x", 128<<10),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		want := assert.New(t)

		output, stats, err := Process(context.Background(), strings.NewReader(input), keepAll)
		if err != nil {
			want.True(errors.Is(err, ErrReadInput), "the only failure is ErrReadInput, got %v", err)
			return
		}

		want.Equal(stats.Total, stats.Kept, "the identity transform keeps every line it sees")

		separators := 0
		if stats.Total > 0 {
			separators = int(stats.Total) - 1
		}
		want.Equal(separators, strings.Count(string(output), "\n"), "lines are joined with exactly one LF between them")
	})
}
