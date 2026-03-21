package binding

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"

	"github.com/MeGaNeKoS/neoma/core"
)


// ContextError represents an HTTP error with a status code, message, and
// optional underlying errors.
type ContextError struct {
	Code int
	Msg  string
	Errs []error
}

// Error returns the error message string.
func (e *ContextError) Error() string {
	return e.Msg
}

// IntoUnmarshaler is a function type that deserializes bytes into a value.
type IntoUnmarshaler = func(data []byte, v any) error

// BodyProcessingConfig holds the parameters needed to validate and unmarshal
// a request body into the target input struct.
type BodyProcessingConfig struct {
	Body           []byte
	Op             core.Operation
	Value          reflect.Value
	HasInputBody   bool
	InputBodyIndex []int
	Unmarshaler    IntoUnmarshaler
	Validator      func(data any, res *core.ValidateResult)
	Defaults       *FindResult[any]
	Result         *core.ValidateResult
}


// ReadBody reads the request body from ctx into buf, enforcing the given
// maxBytes limit. It returns a ContextError on timeout or size violation.
func ReadBody(buf io.Writer, ctx core.Context, maxBytes int64) *ContextError {
	reader := ctx.BodyReader()
	if reader == nil {
		reader = bytes.NewReader(nil)
	}
	if closer, ok := reader.(io.Closer); ok {
		defer func() { _ = closer.Close() }()
	}
	if maxBytes > 0 {
		reader = io.LimitReader(reader, maxBytes)
	}
	count, err := io.Copy(buf, reader)
	if maxBytes > 0 {
		if count == maxBytes {
			return &ContextError{
				Code: http.StatusRequestEntityTooLarge,
				Msg:  fmt.Sprintf("request body is too large limit=%d bytes", maxBytes),
			}
		}
	}
	if err != nil {
		var nErr net.Error
		if errors.As(err, &nErr) && nErr.Timeout() {
			return &ContextError{Code: http.StatusRequestTimeout, Msg: "request body read timeout"}
		}
		return &ContextError{
			Code: http.StatusInternalServerError,
			Msg:  "cannot read request body",
			Errs: []error{err},
		}
	}
	return nil
}

// ProcessRegularMsgBody validates and unmarshals a request body according to
// the provided configuration. It returns the error HTTP status code (or -1 if
// none) and any ContextError encountered.
func ProcessRegularMsgBody(cfg BodyProcessingConfig) (int, *ContextError) {
	errStatus := -1
	if len(cfg.Body) == 0 {
		if cfg.Op.RequestBody != nil && cfg.Op.RequestBody.Required {
			return errStatus, &ContextError{Code: http.StatusBadRequest, Msg: "request body is required"}
		}
		return errStatus, nil
	}
	if !cfg.HasInputBody {
		return errStatus, nil
	}

	isValid := true
	if !cfg.Op.SkipValidateBody {
		validateErrStatus := validateBody(cfg.Body, cfg.Unmarshaler, cfg.Validator, cfg.Result)
		errStatus = validateErrStatus
		if errStatus > 0 {
			isValid = false
		}
	}

	if len(cfg.InputBodyIndex) > 0 {
		if err := parseBodyInto(cfg.Value, cfg.InputBodyIndex, cfg.Unmarshaler, cfg.Body, cfg.Defaults); err != nil && isValid {
			// Validation passed but unmarshaling into the concrete type failed.
			// This can happen when the validator operates on a generic
			// representation (map/slice) that does not catch type-level
			// incompatibilities with the target struct.
			cfg.Result.Errors = append(cfg.Result.Errors, err)
		}
	}
	return errStatus, nil
}


func parseBodyInto(v reflect.Value, bodyIndex []int, u IntoUnmarshaler, body []byte, defaults *FindResult[any]) *core.ErrorDetail {
	// We need to get the body into the correct type now that it has been
	// validated. Benchmarks on Go 1.20 show that using json.Unmarshal a
	// second time is faster than mapstructure.Decode or any of the other
	// common reflection-based approaches when using real-world medium-sized
	// JSON payloads with lots of strings.
	f := v.FieldByIndex(bodyIndex)
	if err := u(body, f.Addr().Interface()); err != nil {
		return &core.ErrorDetail{
			Location: "body",
			Message:  err.Error(),
			Value:    string(body),
		}
	}
	defaults.Every(v, func(item reflect.Value, def any) {
		if item.IsZero() {
			if item.Kind() == reflect.Pointer {
				item.Set(reflect.New(item.Type().Elem()))
				item = item.Elem()
			}
			item.Set(reflect.Indirect(reflect.ValueOf(def)))
		}
	})
	return nil
}

func validateBody(body []byte, u IntoUnmarshaler, validator func(data any, res *core.ValidateResult), res *core.ValidateResult) int {
	errStatus := -1
	// Validate the input. First, parse the body into []any or map[string]any
	// or equivalent, which can be easily validated. Then, convert to the
	// expected struct type to call the handler.
	var parsed any
	if err := u(body, &parsed); err != nil {
		errStatus = http.StatusBadRequest
		if errors.Is(err, core.ErrUnknownContentType) {
			errStatus = http.StatusUnsupportedMediaType
		}
		res.Errors = append(res.Errors, &core.ErrorDetail{
			Location: "body",
			Message:  err.Error(),
			Value:    string(body),
		})
	} else {
		preValidationErrCount := len(res.Errors)
		validator(parsed, res)
		if len(res.Errors)-preValidationErrCount > 0 {
			errStatus = http.StatusUnprocessableEntity
		}
	}
	return errStatus
}
