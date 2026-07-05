package lines

import (
	errs "github.com/gomatic/go-error"
)

// ErrReadInput is returned when reading the input fails. The underlying cause
// is wrapped via errs.Const.With, so the result matches both ErrReadInput and
// the cause under errors.Is.
const ErrReadInput errs.Const = "failed to read input"
