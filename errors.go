package lines

// Error is the package sentinel-error type. Every error this package can emit is
// declared as a const of this type so each path is matchable with errors.Is
// instead of by string comparison.
type Error string

// Error returns the constant's text, implementing the error interface.
func (e Error) Error() string { return string(e) }

// ErrReadInput is returned when reading the input fails. The underlying cause is
// wrapped with %w, so the result matches both ErrReadInput and the cause under
// errors.Is.
const ErrReadInput Error = "failed to read input"
