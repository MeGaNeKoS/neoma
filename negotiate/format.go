package negotiate

import (
	"encoding/json"
	"io"

	"github.com/MeGaNeKoS/neoma/core"
)

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

func DefaultFormats() map[string]core.Format {
	f := DefaultJSONFormat()
	return map[string]core.Format{
		"application/json": f,
		"json":             f,
	}
}
