package cbor

import (
	"io"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/fxamacker/cbor/v2"
)

var cborEncMode, _ = cbor.EncOptions{
	Sort:          cbor.SortCanonical,
	ShortestFloat: cbor.ShortestFloat16,
	NaNConvert:    cbor.NaNConvert7e00,
	InfConvert:    cbor.InfConvertFloat16,
	IndefLength:   cbor.IndefLengthForbidden,
	Time:          cbor.TimeUnixDynamic,
	TimeTag:       cbor.EncTagRequired,
}.EncMode()

var DefaultCBORFormat = core.Format{
	Marshal: func(w io.Writer, v any) error {
		return cborEncMode.NewEncoder(w).Encode(v)
	},
	Unmarshal: cbor.Unmarshal,
}
