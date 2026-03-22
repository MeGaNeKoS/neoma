// Package main demonstrates how to build a REST API with neoma, showcasing
// input/output struct wiring, route groups, middleware, security, tagging,
// hidden operations, file uploads, and error handling.
//
// Run:
//
//	go run ./examples/crud
//
// Then visit http://localhost:8888/public/docs for the interactive API docs.
package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/MeGaNeKoS/neoma/adapters/neomastdlib"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/errors"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
)

// ---------------------------------------------------------------------------
// Domain model
// ---------------------------------------------------------------------------

type Item struct {
	ID   int    `json:"id" example:"1"`
	Name string `json:"name" example:"Widget"`
}

// ---------------------------------------------------------------------------
// In-memory store
// ---------------------------------------------------------------------------

var (
	mu    sync.Mutex
	items = map[int]*Item{}
	seq   = 0
)

// ---------------------------------------------------------------------------
// Input structs define how neoma reads the HTTP request.
//
//	path:"x"              → from URL path parameter {x}
//	query:"x"             → from query string ?x=...
//	header:"X"            → from request header
//	Body T                → parsed from the JSON request body
//	Body struct{ F FormFile `form:"f"` } → multipart form file
//
// Validation tags: required, minLength, maxLength, minimum, maximum, enum, etc.
// Documentation tags: doc, example, default, deprecated
// ---------------------------------------------------------------------------

type CreateInput struct {
	Body struct {
		Name string `json:"name" required:"true" minLength:"1" doc:"Item name"`
	}
}

type IDPath struct {
	ID int `path:"id" example:"1" doc:"Item ID"`
}

type ListInput struct {
	Limit int `query:"limit" minimum:"1" maximum:"100" default:"20" doc:"Max items to return"`
}

type UpdateInput struct {
	ID   int `path:"id" example:"1"`
	Body struct {
		Name string `json:"name" required:"true" minLength:"1"`
	}
}

type UploadInput struct {
	Body struct {
		File core.FormFile `form:"file" required:"true" doc:"File to upload"`
	}
}

// ---------------------------------------------------------------------------
// Output structs define how neoma writes the HTTP response.
//
//	Status int              `yaml:"-"`           → HTTP status code
//	Location string         `header:"Location"`  → response header
//	Body T                                       → serialized as JSON body
//
// Return nil output for no-body responses (e.g. 204 after delete).
// Return nil, err to trigger the ErrorHandler envelope.
// ---------------------------------------------------------------------------

type ItemOutput struct {
	Status   int    `yaml:"-"`
	Location string `header:"Location"`
	Body     *Item
}

type ListOutput struct {
	Body []*Item
}

type MessageOutput struct {
	Body struct {
		Message string `json:"message" example:"ok"`
	}
}

type HealthOutput struct {
	Body struct {
		Status string `json:"status" example:"healthy"`
	}
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func listItems(_ context.Context, in *ListInput) (*ListOutput, error) {
	mu.Lock()
	defer mu.Unlock()

	out := make([]*Item, 0, len(items))
	for _, item := range items {
		out = append(out, item)
		if len(out) >= in.Limit {
			break
		}
	}
	return &ListOutput{Body: out}, nil
}

func createItem(_ context.Context, in *CreateInput) (*ItemOutput, error) {
	mu.Lock()
	defer mu.Unlock()

	seq++
	item := &Item{ID: seq, Name: in.Body.Name}
	items[seq] = item

	return &ItemOutput{
		Status:   http.StatusCreated,
		Location: fmt.Sprintf("/items/%d", item.ID),
		Body:     item,
	}, nil
}

func getItem(_ context.Context, in *IDPath) (*ItemOutput, error) {
	mu.Lock()
	defer mu.Unlock()

	item, ok := items[in.ID]
	if !ok {
		return nil, errors.ErrorNotFound(fmt.Sprintf("item %d not found", in.ID))
	}
	return &ItemOutput{Body: item}, nil
}

func updateItem(_ context.Context, in *UpdateInput) (*ItemOutput, error) {
	mu.Lock()
	defer mu.Unlock()

	item, ok := items[in.ID]
	if !ok {
		return nil, errors.ErrorNotFound(fmt.Sprintf("item %d not found", in.ID))
	}
	item.Name = in.Body.Name
	return &ItemOutput{Body: item}, nil
}

func deleteItem(_ context.Context, in *IDPath) (*struct{}, error) {
	mu.Lock()
	defer mu.Unlock()

	if _, ok := items[in.ID]; !ok {
		return nil, errors.ErrorNotFound(fmt.Sprintf("item %d not found", in.ID))
	}
	delete(items, in.ID)
	return nil, nil //nolint:nilnil // nil output = 204 No Content
}

func uploadFile(_ context.Context, in *UploadInput) (*MessageOutput, error) {
	data, err := io.ReadAll(in.Body.File)
	if err != nil {
		return nil, errors.ErrorBadRequest("failed to read file")
	}
	o := &MessageOutput{}
	o.Body.Message = fmt.Sprintf("received %d bytes: %s", len(data), in.Body.File.Filename)
	return o, nil
}

func healthCheck(_ context.Context, _ *struct{}) (*HealthOutput, error) {
	o := &HealthOutput{}
	o.Body.Status = "healthy"
	return o, nil
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

// demoAuthMiddleware is a simple auth middleware that checks for a Bearer token.
// In a real app this would validate JWTs, API keys, etc.
func demoAuthMiddleware(ctx core.Context, next func(core.Context)) {
	auth := ctx.Header("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || len(auth) <= 7 {
		ctx.SetStatus(http.StatusUnauthorized)
		ctx.SetHeader("Content-Type", "application/json")
		_, _ = ctx.BodyWriter().Write([]byte(`{"status":401,"detail":"missing or invalid bearer token"}`))
		return
	}
	next(ctx)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	mux := http.NewServeMux()

	config := neoma.DefaultConfig("CRUD Example", "1.0.0")
	api := neomastdlib.New(mux, config)

	// Root group. All routes registered through this group get the shared
	// middleware and modifiers applied.
	root := middleware.NewGroup(api)

	// --- Public routes (no auth) -------------------------------------------

	// Health check: no group, no tag, no auth.
	neoma.Get[struct{}, HealthOutput](root, "/health", healthCheck)

	// --- Items group (tagged + authenticated) -------------------------------

	// Create a sub-group with prefix. UseDefaultTag auto-tags all operations
	// in this group as "items" in the OpenAPI spec.
	itemsGroup := root.Group("/items")
	itemsGroup.UseDefaultTag("items")

	// WithSecurity registers the bearer scheme in the OpenAPI spec, applies
	// the security requirement to all operations in this group, and adds
	// the auth middleware. One call instead of three manual steps.
	itemsGroup.WithSecurity("bearerAuth", &core.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}, demoAuthMiddleware)

	neoma.Get[ListInput, ListOutput](itemsGroup, "/", listItems)
	neoma.Post[CreateInput, ItemOutput](itemsGroup, "/", createItem)
	neoma.Get[IDPath, ItemOutput](itemsGroup, "/{id}", getItem)
	neoma.Put[UpdateInput, ItemOutput](itemsGroup, "/{id}", updateItem)
	neoma.Delete[IDPath, struct{}](itemsGroup, "/{id}", deleteItem)

	// --- Files group (tagged, no auth) --------------------------------------

	filesGroup := root.Group("/files")
	filesGroup.UseDefaultTag("files")

	neoma.Post[UploadInput, MessageOutput](filesGroup, "/upload", uploadFile)

	// --- Hidden operation (internal spec only) ------------------------------

	// Hidden operations are routed normally but excluded from the public
	// OpenAPI spec. They appear in /internal/openapi.json if configured.
	neoma.Get[struct{}, HealthOutput](root, "/debug/info", healthCheck,
		func(op *core.Operation) {
			op.Hidden = true
		},
	)

	// --- Custom operation options -------------------------------------------

	// Operations can be customized with modifiers: tags, errors, summary, etc.
	neoma.Get[struct{}, MessageOutput](root, "/ping", func(_ context.Context, _ *struct{}) (*MessageOutput, error) {
		o := &MessageOutput{}
		o.Body.Message = "pong"
		return o, nil
	}, func(op *core.Operation) {
		op.Summary = "Ping the server"
		op.Tags = []string{"system"}
		op.Deprecated = true
	})

	fmt.Println("Listening on http://localhost:8888")
	fmt.Println("Docs at    http://localhost:8888/public/docs")

	if err := http.ListenAndServe(":8888", mux); err != nil {
		panic(err)
	}
}

// ---------------------------------------------------------------------------
// Example requests:
//
//   # Health (public, no auth)
//   curl localhost:8888/health
//   → 200 {"status":"healthy"}
//
//   # List items (requires auth)
//   curl localhost:8888/items -H "Authorization: Bearer mytoken"
//   → 200 [...]
//
//   # Create item
//   curl -X POST localhost:8888/items \
//        -H "Authorization: Bearer mytoken" \
//        -H "Content-Type: application/json" \
//        -d '{"name":"Widget"}'
//   → 201 + Location: /items/1
//
//   # Upload file (no auth)
//   curl -X POST localhost:8888/files/upload \
//        -F "file=@README.md"
//   → 200 {"message":"received 1234 bytes: README.md"}
//
//   # Missing auth
//   curl localhost:8888/items
//   → 401 {"status":401,"detail":"missing or invalid bearer token"}
//
//   # Validation error
//   curl -X POST localhost:8888/items \
//        -H "Authorization: Bearer mytoken" \
//        -H "Content-Type: application/json" \
//        -d '{"name":""}'
//   → 422 {"status":422,"detail":"validation failed"}
//
//   # Not found
//   curl localhost:8888/items/999 -H "Authorization: Bearer mytoken"
//   → 404 {"status":404,"detail":"item 999 not found"}
//
//   # Deprecated ping
//   curl localhost:8888/ping
//   → 200 {"message":"pong"}
// ---------------------------------------------------------------------------
