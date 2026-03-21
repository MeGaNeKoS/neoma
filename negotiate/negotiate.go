package negotiate

import (
	"fmt"
	"io"
	"strings"

	"github.com/MeGaNeKoS/neoma/core"
)

// Negotiator selects the appropriate format for marshalling and unmarshalling
// based on HTTP content type headers.
type Negotiator struct {
	formats        map[string]core.Format
	formatKeys     []string
	defaultFormat  string
	noFallback     bool
}

// NewNegotiator creates a Negotiator with the given format map, default content
// type, and fallback behavior. When noFallback is true, unrecognized Accept
// values produce an error instead of falling back to the default format.
func NewNegotiator(formats map[string]core.Format, defaultFormat string, noFallback bool) *Negotiator {
	n := &Negotiator{
		formats:       make(map[string]core.Format, len(formats)),
		defaultFormat: defaultFormat,
		noFallback:    noFallback,
	}

	// Place the default format first so it wins ties in SelectQValueFast.
	if defaultFormat != "" {
		n.formatKeys = append(n.formatKeys, defaultFormat)
	}

	for k, v := range formats {
		n.formats[k] = v
		if k != defaultFormat {
			n.formatKeys = append(n.formatKeys, k)
		}
	}

	return n
}

// Negotiate selects the best content type from the Accept header value,
// returning the matched content type or an error if no match is found.
func (n *Negotiator) Negotiate(accept string) (string, error) {
	ct := SelectQValueFast(accept, n.formatKeys)
	if ct == "" {
		if n.noFallback {
			return "", core.ErrUnknownAcceptContentType
		}
		if len(n.formatKeys) > 0 {
			ct = n.formatKeys[0]
		}
	}
	if _, ok := n.formats[ct]; !ok {
		return ct, fmt.Errorf("%w: %s", core.ErrUnknownContentType, ct)
	}
	return ct, nil
}

// Marshal encodes the value v into the writer using the format associated with
// the given content type.
func (n *Negotiator) Marshal(w io.Writer, ct string, v any) error {
	f, ok := n.formats[ct]
	if !ok {
		start, end, err := parseContentType(ct)
		if err != nil {
			return err
		}
		f, ok = n.formats[ct[start:end]]
	}
	if !ok {
		return fmt.Errorf("%w: %s", core.ErrUnknownContentType, ct)
	}
	return f.Marshal(w, v)
}

// Unmarshal decodes the byte slice into v using the format associated with the
// given content type, defaulting to JSON if the content type is empty.
func (n *Negotiator) Unmarshal(ct string, data []byte, v any) error {
	start, end, err := parseContentType(ct)
	if err != nil {
		return err
	}

	resolved := ct[start:end]
	if resolved == "" {
		// Default to JSON since this is an API.
		resolved = "application/json"
	}

	f, ok := n.formats[resolved]
	if !ok {
		return fmt.Errorf("%w: %s", core.ErrUnknownContentType, ct)
	}

	return f.Unmarshal(data, v)
}

func parseContentType(contentType string) (int, int, error) {
	start := strings.IndexRune(contentType, '+') + 1
	end := strings.IndexRune(contentType, ';')
	if end == -1 {
		end = len(contentType)
	}

	if end < start {
		// The '+' appears after the ';', which is malformed.
		return 0, 0, fmt.Errorf("%w: %s", core.ErrUnknownContentType, contentType)
	}

	return start, end, nil
}
