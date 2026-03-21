package core

import "fmt"

// ValidateMode controls which validation rules apply, distinguishing between
// data read from the server (responses) and data written to the server
// (requests).
type ValidateMode int

const (
	// ModeReadFromServer applies validation rules for server responses.
	ModeReadFromServer ValidateMode = iota

	// ModeWriteToServer applies validation rules for client requests.
	ModeWriteToServer
)

// ValidateResult collects validation errors produced during schema validation.
type ValidateResult struct {
	Errors []error
}

// Add appends a validation error with the given path, offending value, and
// message.
func (r *ValidateResult) Add(path *PathBuffer, v any, msg string) {
	r.Errors = append(r.Errors, &ErrorDetail{
		Message:  msg,
		Location: path.String(),
		Value:    v,
	})
}

// Addf appends a validation error using a fmt.Sprintf-style format string.
func (r *ValidateResult) Addf(path *PathBuffer, v any, format string, args ...any) {
	r.Errors = append(r.Errors, &ErrorDetail{
		Message:  fmt.Sprintf(format, args...),
		Location: path.String(),
		Value:    v,
	})
}

// Reset clears all collected errors, allowing the result to be reused.
func (r *ValidateResult) Reset() {
	r.Errors = r.Errors[:0]
}

// HasErrors reports whether any validation errors have been collected.
func (r *ValidateResult) HasErrors() bool {
	return len(r.Errors) > 0
}
