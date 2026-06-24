package goapitomcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPHandlerListsAndCallsOpenAPITools(t *testing.T) {
	ctx := context.Background()
	var sawAuth bool
	var sawPathQuery bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/override/pets/abc" && r.URL.Query().Get("verbose") == "true" {
			sawPathQuery = true
		}
		if r.Header.Get("Authorization") == "Bearer test-token" {
			sawAuth = true
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      "abc",
			"verbose": r.URL.Query().Get("verbose"),
		})
	}))
	defer backend.Close()

	spec := `
openapi: 3.1.0
info:
  title: Runtime API
  version: 2.0.0
servers:
  - url: https://spec-server.example.invalid/base
paths:
  /pets/{id}:
    get:
      operationId: getPet
      summary: Get pet
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
        - name: verbose
          in: query
          schema:
            type: boolean
      responses:
        '200':
          description: ok
`
	baseURL := mustParseURL(t, backend.URL+"/override")
	handler, err := NewHandler(ctx, Config{
		Spec:    strings.NewReader(spec),
		BaseURL: baseURL,
		BeforeRequest: func(_ context.Context, op OperationContext, r *http.Request) error {
			if op.ToolName != "getPet" {
				t.Fatalf("OperationContext.ToolName = %q, want getPet", op.ToolName)
			}
			r.Header.Set("Authorization", "Bearer test-token")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	mcpServer := httptest.NewServer(handler)
	defer mcpServer.Close()

	session := connectMCP(t, ctx, mcpServer.URL)
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "getPet" {
		t.Fatalf("tools = %#v, want getPet", tools.Tools)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "getPet",
		Arguments: map[string]any{
			"path": map[string]any{"id": "abc"},
			"query": map[string]any{
				"verbose": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() IsError = true, content = %#v", result.Content)
	}
	payload := decodePayload(t, result)
	if payload.Status != 200 {
		t.Fatalf("payload.Status = %d, want 200", payload.Status)
	}
	body := payload.Body.(map[string]any)
	if body["id"] != "abc" || body["verbose"] != "true" {
		t.Fatalf("payload.Body = %#v", payload.Body)
	}
	if !sawPathQuery {
		t.Fatalf("backend did not see BaseURL override path and serialized query")
	}
	if !sawAuth {
		t.Fatalf("backend did not see BeforeRequest auth header")
	}
}

func TestCallOperationPostsJSONBody(t *testing.T) {
	var gotBody map[string]any
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/pets" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type = %q", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer backend.Close()

	spec := `
openapi: 3.0.3
info:
  title: Runtime API
  version: 1.0.0
paths:
  /pets:
    post:
      operationId: createPet
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        '201':
          description: created
`
	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:    strings.NewReader(spec),
		BaseURL: mustParseURL(t, backend.URL),
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	result, err := catalog.callOperation(context.Background(), findOperation(t, catalog, "POST", "/pets"), callRequest(map[string]any{
		"body": map[string]any{"name": "Milo"},
	}))
	if err != nil {
		t.Fatalf("callOperation() protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("callOperation() IsError = true")
	}
	if gotBody["name"] != "Milo" {
		t.Fatalf("gotBody = %#v", gotBody)
	}
	if got := decodePayload(t, result).Status; got != http.StatusCreated {
		t.Fatalf("status = %d, want 201", got)
	}
}

func TestCallOperationAppliesStaticHeadersSecurityDefaultsAndFormBody(t *testing.T) {
	var gotForm url.Values
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Static") != "configured" {
			t.Fatalf("X-Static = %q, want configured", r.Header.Get("X-Static"))
		}
		if r.Header.Get("X-API-Key") != "secret-key" {
			t.Fatalf("X-API-Key = %q, want secret-key", r.Header.Get("X-API-Key"))
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type = %q", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		gotForm = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer backend.Close()

	spec := `
openapi: 3.0.3
info:
  title: Form API
  version: 1.0.0
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
      x-defaultCredential: secret-key
security:
  - ApiKeyAuth: []
paths:
  /tokens:
    post:
      operationId: createToken
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [username]
              properties:
                username:
                  type: string
                scopes:
                  type: array
                  items:
                    type: string
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  ok:
                    type: boolean
`
	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:                strings.NewReader(spec),
		BaseURL:             mustParseURL(t, backend.URL),
		StaticHeaders:       http.Header{"X-Static": []string{"configured"}},
		UseSecurityDefaults: true,
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	result, err := catalog.callOperation(context.Background(), catalog.Operations[0], callRequest(map[string]any{
		"body": map[string]any{
			"username": "alex",
			"scopes":   []any{"read", "write"},
		},
	}))
	if err != nil {
		t.Fatalf("callOperation() protocol error = %v", err)
	}
	if result.IsError {
		t.Fatalf("callOperation() IsError = true")
	}
	if gotForm.Get("username") != "alex" {
		t.Fatalf("username form value = %q", gotForm.Get("username"))
	}
	if values := gotForm["scopes"]; len(values) != 2 || values[0] != "read" || values[1] != "write" {
		t.Fatalf("scopes form values = %#v", values)
	}
}

func TestCallOperationMarksNon2xxAsToolError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	defer backend.Close()

	spec := simpleGetSpec()
	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:    strings.NewReader(spec),
		BaseURL: mustParseURL(t, backend.URL),
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	result, err := catalog.callOperation(context.Background(), catalog.Operations[0], callRequest(nil))
	if err != nil {
		t.Fatalf("callOperation() protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("IsError = false, want true")
	}
	if got := decodePayload(t, result).Status; got != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", got)
	}
}

func TestHandlerCanBeMountedOnExistingMux(t *testing.T) {
	ctx := context.Background()
	handler, err := NewHandler(ctx, Config{
		Spec:    strings.NewReader(simpleGetSpec()),
		BaseURL: mustParseURL(t, "https://api.example.test"),
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	session := connectMCP(t, ctx, server.URL+"/mcp")
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "getStatus" {
		t.Fatalf("tools = %#v, want getStatus", tools.Tools)
	}
}

func TestNewHandlerFromFileCreatesMountableMCPHandler(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	writeFile(t, path, simpleGetSpec())

	handler, err := NewHandlerFromFile(ctx, path, Config{
		BaseURL: mustParseURL(t, "https://api.example.test"),
	})
	if err != nil {
		t.Fatalf("NewHandlerFromFile() error = %v", err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	session := connectMCP(t, ctx, server.URL)
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "getStatus" {
		t.Fatalf("tools = %#v, want getStatus", tools.Tools)
	}
}

func TestWithCORSHandlesPreflightAndPassesRequests(t *testing.T) {
	var served bool
	handler := WithCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
		w.WriteHeader(http.StatusAccepted)
	}), CORSOptions{
		AllowedOrigins: []string{"https://app.example.test"},
	})

	preflight := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	preflight.Header.Set("Origin", "https://app.example.test")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodPost)
	preflightRecorder := httptest.NewRecorder()
	handler.ServeHTTP(preflightRecorder, preflight)

	if preflightRecorder.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", preflightRecorder.Code)
	}
	if served {
		t.Fatalf("preflight reached wrapped handler")
	}
	if got := preflightRecorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.test" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := preflightRecorder.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf("Access-Control-Allow-Methods = %q, want POST", got)
	}

	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader("{}"))
	request.Header.Set("Origin", "https://app.example.test")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("request status = %d, want 202", recorder.Code)
	}
	if !served {
		t.Fatalf("request did not reach wrapped handler")
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.test" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
}

func TestCallOperationRejectsUnsupportedBodyMediaType(t *testing.T) {
	spec := `
openapi: 3.0.3
info:
  title: Runtime API
  version: 1.0.0
paths:
  /messages:
    post:
      operationId: createMessage
      requestBody:
        required: true
        content:
          text/plain:
            schema:
              type: string
      responses:
        '201':
          description: created
`
	catalog, err := LoadCatalog(context.Background(), Config{
		Spec:    strings.NewReader(spec),
		BaseURL: mustParseURL(t, "https://api.example.test"),
	})
	if err != nil {
		t.Fatalf("LoadCatalog() error = %v", err)
	}
	result, err := catalog.callOperation(context.Background(), catalog.Operations[0], callRequest(map[string]any{
		"body": "hello",
	}))
	if err != nil {
		t.Fatalf("callOperation() protocol error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("IsError = false, want true")
	}
	payload := decodePayload(t, result)
	if !strings.Contains(payload.Body.(string), "supported JSON or form request body") {
		t.Fatalf("payload.Body = %#v", payload.Body)
	}
}

func connectMCP(t *testing.T, ctx context.Context, endpoint string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint}, nil)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	return session
}

func callRequest(args map[string]any) *mcp.CallToolRequest {
	var raw json.RawMessage
	if args != nil {
		raw, _ = json.Marshal(args)
	}
	return &mcp.CallToolRequest{
		Params: &mcp.CallToolParamsRaw{Arguments: raw},
	}
}

func decodePayload(t *testing.T, result *mcp.CallToolResult) toolPayload {
	t.Helper()
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal StructuredContent: %v", err)
	}
	var payload toolPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal StructuredContent: %v", err)
	}
	return payload
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func simpleGetSpec() string {
	return `
openapi: 3.0.3
info:
  title: Runtime API
  version: 1.0.0
paths:
  /status:
    get:
      operationId: getStatus
      responses:
        '200':
          description: ok
`
}
