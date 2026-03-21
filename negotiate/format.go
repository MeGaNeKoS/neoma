// Package negotiate implements HTTP content negotiation for selecting request
// and response formats based on Accept and Content-Type headers.
package negotiate

import (
	"encoding/json"
	"io"

	"github.com/MeGaNeKoS/neoma/core"
)

// DefaultJSONFormat returns a Format that uses the standard library JSON
// encoder and decoder with HTML escaping disabled.
func DefaultJSONFormat() core.Format {
	return core.Format{
		Marshal: func(w io.Writer, v any) error {
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			return enc.Encode(v)
		},
		Unmarshal: json.Unmarshal,
	}
}

// DefaultFormats returns a format map with JSON registered under both
// "application/json" and "json" content types.
func DefaultFormats() map[string]core.Format {
	f := DefaultJSONFormat()
	return map[string]core.Format{
		"application/json": f,
		"json":             f,
	}
}
