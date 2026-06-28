# go-lines

Line-oriented text processing for Go: small pure per-line operations (`Uppercase`, `WithPrefix`, `Numbered`, `Contains`) plus a streaming `Process` that applies a `Transform` to each line of an `io.Reader`, honors context cancellation, and reports line counts.

## Install

```sh
go get github.com/gomatic/go-lines
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gomatic/go-lines"
)

func main() {
	// A transform keeps lines containing "go" and uppercases them with a number.
	transform := func(line lines.Line, n lines.LineNumber) (lines.Line, bool) {
		if !lines.Contains(line, "go") {
			return "", false
		}
		return lines.Numbered(lines.Uppercase(line), n), true
	}

	out, stats, err := lines.Process(context.Background(), strings.NewReader("go fast\nstop\ngo home"), transform)
	if err != nil {
		panic(err)
	}
	fmt.Println(out)                       //    1 | GO FAST\n   3 | GO HOME
	fmt.Println(stats.Total, stats.Kept)   // 3 2
}
```

`Process` returns `ErrReadInput` (matchable with `errors.Is`, wrapping the underlying cause) when the reader fails. The package is CLI-agnostic and dependency-free (testify for tests only).
