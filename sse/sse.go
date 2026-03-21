// Package sse provides Server-Sent Events (SSE) support for streaming
// responses from neoma API handlers.
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
)

// Message represents a single Server-Sent Event with optional ID, data payload,
// and retry interval.
type Message struct {
	ID    int
	Data  any
	Retry int
}

// Sender is a function that sends a Server-Sent Event message to the client.
type Sender func(Message) error

// Data is a convenience method that sends a message containing only the given
// data payload.
func (s Sender) Data(data any) error {
	return s(Message{Data: data})
}

// WriteTimeout is the deadline applied to each SSE write operation.
var WriteTimeout = 5 * time.Second

// Register adds an SSE endpoint to the API. It configures the operation's
// response schema from the eventTypeMap and streams events via the handler's
// Sender callback.
func Register[I any](api core.API, op core.Operation, eventTypeMap map[string]any, handler func(ctx context.Context, input *I, send Sender)) {
	if op.Responses == nil {
		op.Responses = map[string]*core.Response{}
	}
	if op.Responses["200"] == nil {
		op.Responses["200"] = &core.Response{}
	}
	if op.Responses["200"].Content == nil {
		op.Responses["200"].Content = map[string]*core.MediaType{}
	}

	typeToEvent := make(map[reflect.Type]string, len(eventTypeMap))
	dataSchemas := make([]*core.Schema, 0, len(eventTypeMap))
	for k, v := range eventTypeMap {
		vt := core.Deref(reflect.TypeOf(v))
		typeToEvent[vt] = k
		required := []string{"data"}
		if k != "" && k != "message" {
			required = append(required, "event")
		}
		s := &core.Schema{
			Title: "Event " + k,
			Type:  core.TypeObject,
			Properties: map[string]*core.Schema{
				"id":    {Type: core.TypeInteger},
				"event": {Type: core.TypeString, Extensions: map[string]any{"const": k}},
				"data":  api.OpenAPI().Components.Schemas.Schema(vt, true, k),
				"retry": {Type: core.TypeInteger},
			},
			Required: required,
		}
		dataSchemas = append(dataSchemas, s)
	}

	slices.SortFunc(dataSchemas, func(a, b *core.Schema) int {
		return strings.Compare(a.Title, b.Title)
	})

	op.Responses["200"].Content["text/event-stream"] = &core.MediaType{
		Schema: &core.Schema{
			Type: core.TypeArray,
			Items: &core.Schema{
				Extensions: map[string]any{"oneOf": dataSchemas},
			},
		},
	}

	neoma.Register(api, op, func(ctx context.Context, input *I) (*core.StreamResponse, error) {
		return &core.StreamResponse{
			Body: func(ctx core.Context, _ core.API) {
				ctx.SetHeader("Content-Type", "text/event-stream")
				bw := ctx.BodyWriter()
				encoder := json.NewEncoder(bw)

				flusher := findInterface[http.Flusher](bw)
				deadliner := findInterface[writeDeadliner](bw)

				send := func(msg Message) error {
					if deadliner != nil {
						_ = (*deadliner).SetWriteDeadline(time.Now().Add(WriteTimeout))
					}

					if msg.ID > 0 {
						_, _ = bw.Write(fmt.Appendf(nil, "id: %d\n", msg.ID))
					}
					if msg.Retry > 0 {
						_, _ = bw.Write(fmt.Appendf(nil, "retry: %d\n", msg.Retry))
					}

					event := typeToEvent[core.Deref(reflect.TypeOf(msg.Data))]
					if event != "" && event != "message" {
						_, _ = bw.Write([]byte("event: " + event + "\n"))
					}

					if _, err := bw.Write([]byte("data: ")); err != nil {
						return err
					}
					if err := encoder.Encode(msg.Data); err != nil {
						return err
					}
					_, _ = bw.Write([]byte("\n"))

					if flusher != nil {
						(*flusher).Flush()
					}
					return nil
				}

				handler(ctx.Context(), input, send)
			},
		}, nil
	})
}

type writeDeadliner interface {
	SetWriteDeadline(time.Time) error
}

func findInterface[T any](w io.Writer) *T {
	for {
		if v, ok := w.(T); ok {
			return &v
		}
		if u, ok := w.(interface{ Unwrap() http.ResponseWriter }); ok {
			w = u.Unwrap()
		} else {
			return nil
		}
	}
}
