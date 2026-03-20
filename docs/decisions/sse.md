# Server-Sent Events (SSE)

## What It Is

SSE is a protocol for pushing real-time updates from server to client over a single HTTP connection. Unlike WebSocket, it is unidirectional (server to client only), works through HTTP proxies, and auto-reconnects natively in browsers.

## How Users Use It

```go
import "github.com/MeGaNeKoS/neoma/sse"

type ChatMessage struct {
    User string `json:"user"`
    Text string `json:"text"`
}

type UserJoined struct {
    User string `json:"user"`
}

sse.Register(api, core.Operation{
    Method: http.MethodGet,
    Path:   "/events",
}, map[string]any{
    "message": ChatMessage{},
    "joined":  UserJoined{},
}, func(ctx context.Context, input *struct{}, send sse.Sender) {
    // Runs for the lifetime of the connection.
    send(sse.Message{Data: UserJoined{User: "alice"}})

    for {
        select {
        case <-ctx.Done():
            return
        case msg := <-chatChannel:
            send(sse.Message{Data: ChatMessage{User: msg.User, Text: msg.Text}})
        }
    }
})
```

The `eventTypeMap` maps event names to Go types. This generates the OpenAPI schema for the `text/event-stream` response, documenting each event type.

### Client Side (Browser)

```js
const es = new EventSource("/events");
es.addEventListener("message", (e) => console.log(JSON.parse(e.data)));
es.addEventListener("joined", (e) => console.log(JSON.parse(e.data)));
es.onerror = () => console.log("reconnecting...");
```

No library required. `EventSource` is built into all browsers. For non-browser clients, any HTTP client that reads a streaming response body works.

### Message Fields

| Field   | Purpose |
|---------|---------|
| `ID`    | Monotonic event ID. Clients send `Last-Event-ID` header on reconnect so the server can resume. |
| `Data`  | The payload. Must match one of the types in `eventTypeMap`. Serialized as JSON. |
| `Retry` | Tells the client how long (ms) to wait before reconnecting after a disconnect. |

### Convenience

```go
// Send just data (no ID, no retry)
send.Data(ChatMessage{User: "bob", Text: "hello"})
```

## What The Framework Handles

- `Content-Type: text/event-stream` header
- SSE wire format (`id:`, `event:`, `data:`, `retry:` fields, double newline terminator)
- Auto-flush after each message (unwraps the writer to find `http.Flusher`)
- Per-message write deadline (default 5s) to detect dead connections
- OpenAPI schema generation from the `eventTypeMap`, documenting each event type as a `oneOf` in the response

## Wire Format

What actually goes over the wire for each `send()` call:

```
id: 42
event: joined
data: {"user":"alice"}

```

The `event` field is omitted for the default "message" event type. Each message ends with a blank line (`\n\n`).
