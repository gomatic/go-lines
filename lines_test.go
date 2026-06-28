package lines

import (
	"context"
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
