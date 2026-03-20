package sse_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/MeGaNeKoS/neoma/neomatest"
	"github.com/MeGaNeKoS/neoma/sse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ChatMessage struct {
	Text string `json:"text"`
}

type StatusUpdate struct {
	Online bool `json:"online"`
}

func TestRegisterAndStream(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	sse.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/events",
		OperationID: "stream-events",
	}, map[string]any{
		"chat":   ChatMessage{},
		"status": StatusUpdate{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		_ = send(sse.Message{ID: 1, Data: ChatMessage{Text: "hello"}})
		_ = send.Data(StatusUpdate{Online: true})
		_ = send(sse.Message{ID: 3, Data: ChatMessage{Text: "bye"}, Retry: 5000})
	})

	resp := api.Do(http.MethodGet, "/events")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "text/event-stream", resp.Header().Get("Content-Type"))

	body := resp.Body.String()
	assert.Contains(t, body, "id: 1")
	assert.Contains(t, body, "event: chat")
	assert.Contains(t, body, `"text":"hello"`)
	assert.Contains(t, body, "event: status")
	assert.Contains(t, body, `"online":true`)
	assert.Contains(t, body, "id: 3")
	assert.Contains(t, body, "retry: 5000")
}

func TestRegisterDefaultEventNoEventField(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	sse.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/default-events",
		OperationID: "default-events",
	}, map[string]any{
		"message": ChatMessage{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		_ = send.Data(ChatMessage{Text: "default"})
	})

	resp := api.Do(http.MethodGet, "/default-events")
	assert.Equal(t, http.StatusOK, resp.Code)

	body := resp.Body.String()
	assert.NotContains(t, body, "event:")
	assert.Contains(t, body, `"text":"default"`)
}

func TestRegisterOpenAPISpec(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	sse.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/sse",
		OperationID: "sse-endpoint",
	}, map[string]any{
		"ping": struct{ Seq int `json:"seq"` }{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {})

	resp := api.Do(http.MethodGet, "/openapi.json")
	require.Equal(t, http.StatusOK, resp.Code)

	var spec map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &spec))

	paths, _ := spec["paths"].(map[string]any)
	require.NotNil(t, paths["/sse"])

	pi, _ := paths["/sse"].(map[string]any)
	get, _ := pi["get"].(map[string]any)
	require.NotNil(t, get)

	responses, _ := get["responses"].(map[string]any)
	resp200, _ := responses["200"].(map[string]any)
	require.NotNil(t, resp200)

	content, _ := resp200["content"].(map[string]any)
	require.NotNil(t, content["text/event-stream"])
}

func TestRegisterWithInputParams(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	type Input struct {
		Channel string `query:"channel"`
	}

	sse.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/stream",
		OperationID: "stream-with-params",
	}, map[string]any{
		"data": ChatMessage{},
	}, func(ctx context.Context, input *Input, send sse.Sender) {
		_ = send.Data(ChatMessage{Text: "channel=" + input.Channel})
	})

	resp := api.Do(http.MethodGet, "/stream?channel=general")
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "channel=general")
}

func TestSenderData(t *testing.T) {
	var sent sse.Message
	s := sse.Sender(func(msg sse.Message) error {
		sent = msg
		return nil
	})

	err := s.Data("hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", sent.Data)
	assert.Equal(t, 0, sent.ID)
}

func TestStreamEventOrder(t *testing.T) {
	_, api := neomatest.New(t, neoma.DefaultConfig("Test", "1.0.0"))

	sse.Register(api, core.Operation{
		Method:      http.MethodGet,
		Path:        "/ordered",
		OperationID: "ordered-events",
	}, map[string]any{
		"message": ChatMessage{},
	}, func(ctx context.Context, input *struct{}, send sse.Sender) {
		for i := 1; i <= 3; i++ {
			_ = send(sse.Message{
				ID:   i,
				Data: ChatMessage{Text: strings.Repeat("x", i)},
			})
		}
	})

	resp := api.Do(http.MethodGet, "/ordered")
	assert.Equal(t, http.StatusOK, resp.Code)

	scanner := bufio.NewScanner(resp.Body)
	ids := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "id:") {
			ids++
		}
	}
	assert.Equal(t, 3, ids)
}
