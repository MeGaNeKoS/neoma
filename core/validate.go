package core

import "fmt"

type ValidateMode int

const (
	ModeReadFromServer ValidateMode = iota
	ModeWriteToServer
)

type ValidateResult struct {
	Errors []error
}

func (r *ValidateResult) Add(path *PathBuffer, v any, msg string) {
	r.Errors = append(r.Errors, &ErrorDetail{
		Message:  msg,
		Location: path.String(),
		Value:    v,
	})
}

func (r *ValidateResult) Addf(path *PathBuffer, v any, format string, args ...any) {
	r.Errors = append(r.Errors, &ErrorDetail{
		Message:  fmt.Sprintf(format, args...),
		Location: path.String(),
		Value:    v,
	})
}

func (r *ValidateResult) Reset() {
	r.Errors = r.Errors[:0]
}

func (r *ValidateResult) HasErrors() bool {
	return len(r.Errors) > 0
}
