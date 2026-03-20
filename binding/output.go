package binding

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
)

var statusStrings = map[int]string{
	100: "100", 101: "101", 102: "102", 103: "103",
	200: "200", 201: "201", 202: "202", 203: "203", 204: "204", 205: "205", 206: "206", 207: "207", 208: "208", 226: "226",
	300: "300", 301: "301", 302: "302", 303: "303", 304: "304", 305: "305", 307: "307", 308: "308",
	400: "400", 401: "401", 402: "402", 403: "403", 404: "404", 405: "405", 406: "406", 407: "407", 408: "408", 409: "409", 410: "410", 411: "411", 412: "412", 413: "413", 414: "414", 415: "415", 416: "416", 417: "417", 418: "418", 421: "421", 422: "422", 423: "423", 424: "424", 425: "425", 426: "426", 428: "428", 429: "429", 431: "431", 451: "451",
	500: "500", 501: "501", 502: "502", 503: "503", 504: "504", 505: "505", 506: "506", 507: "507", 508: "508", 510: "510", 511: "511",
}

type HeaderInfo struct {
	Field      reflect.StructField
	Name       string
	TimeFormat string
}

func WriteHeader(write func(string, string), info *HeaderInfo, f reflect.Value) {
	switch f.Kind() {
	case reflect.String:
		if f.String() == "" {
			return
		}
		write(info.Name, f.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		write(info.Name, strconv.FormatInt(f.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		write(info.Name, strconv.FormatUint(f.Uint(), 10))
	case reflect.Float32, reflect.Float64:
		write(info.Name, strconv.FormatFloat(f.Float(), 'f', -1, 64))
	case reflect.Bool:
		write(info.Name, strconv.FormatBool(f.Bool()))
	default:
		if t, ok := f.Interface().(time.Time); ok && !t.IsZero() {
			write(info.Name, t.Format(info.TimeFormat))
			return
		}

		if f.CanAddr() {
			if s, ok := f.Addr().Interface().(fmt.Stringer); ok {
				write(info.Name, s.String())
				return
			}
		}

		write(info.Name, fmt.Sprintf("%v", f.Interface()))
	}
}

func WriteResponse(api core.API, ctx core.Context, status int, ct string, body any) error {
	if ct == "" {
		var err error
		ct, err = api.Negotiate(ctx.Header("Accept"))
		if err != nil {
			errFactory := api.ErrorHandler()
			notAccept := errFactory.NewErrorWithContext(ctx, http.StatusNotAcceptable, "unable to marshal response", err)
			ct = "application/json"
			var ctf core.ContentTyper
			if errors.As(notAccept, &ctf) {
				ct = ctf.ContentType(ct)
			}

			ctx.SetHeader("Content-Type", ct)
			if e := transformAndWrite(api, ctx, http.StatusNotAcceptable, "application/json", notAccept); e != nil {
				return e
			}

			return err
		}

		if ctf, ok := body.(core.ContentTyper); ok {
			ct = ctf.ContentType(ct)
		}

		ctx.SetHeader("Content-Type", ct)
	}

	return transformAndWrite(api, ctx, status, ct, body)
}

func WriteResponseWithPanic(api core.API, ctx core.Context, status int, ct string, body any) {
	if err := WriteResponse(api, ctx, status, ct, body); err != nil {
		panic(err)
	}
}

func findHeaders[O any]() *FindResult[*HeaderInfo] {
	t := reflect.TypeFor[O]()
	return findInType(t, nil, func(sf reflect.StructField, i []int) *HeaderInfo {
		if sf.Anonymous {
			return nil
		}

		header := sf.Tag.Get("header")
		if header == "" {
			header = sf.Name
		}

		timeFormat := ""
		if sf.Type == timeType {
			timeFormat = http.TimeFormat
			if f := sf.Tag.Get("timeFormat"); f != "" {
				timeFormat = f
			}
		}

		return &HeaderInfo{sf, header, timeFormat}
	}, false, "Status", "Body")
}

func transformAndWrite(api core.API, ctx core.Context, status int, ct string, body any) error {
	statusStr, ok := statusStrings[status]
	if !ok {
		statusStr = strconv.Itoa(status)
	}

	ctx.SetStatus(status)

	tVal, tErr := api.Transform(ctx, statusStr, body)
	if tErr != nil {
		_, _ = ctx.BodyWriter().Write([]byte("error transforming response"))
		return fmt.Errorf("error transforming response for %s %s %d: %w", ctx.Operation().Method, ctx.Operation().Path, status, tErr)
	}

	if status != http.StatusNoContent && status != http.StatusNotModified {
		if mErr := api.Marshal(ctx.BodyWriter(), ct, tVal); mErr != nil {
			if errors.Is(ctx.Context().Err(), context.Canceled) {
				// The client disconnected, so do not bother writing anything.
				// Attempt to set the status in case it will get logged.
				ctx.SetStatus(499)
				return nil
			}
			_, _ = ctx.BodyWriter().Write([]byte("error marshaling response"))
			return fmt.Errorf("error marshaling response for %s %s %d: %w", ctx.Operation().Method, ctx.Operation().Path, status, mErr)
		}
	}

	return nil
}
