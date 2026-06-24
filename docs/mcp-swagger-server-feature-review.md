# mcp-swagger-server v0.6.0 Feature Review

Reviewed project: <https://github.com/liliang-cn/mcp-swagger-server/tree/v0.6.0>

## Implemented In This Library

- API filtering inspired by `APIFilter`: exact paths, wildcard path patterns, operation IDs, HTTP methods, tags, and include-only variants are available through `Config.Filter`.
- File-based constructors inspired by `NewFromSwaggerFile`: `LoadCatalogFromFile`, `NewServerFromFile`, and `NewHandlerFromFile` open local OpenAPI files and set a `file://` `SpecBaseURI` for relative refs.
- CORS support inspired by the HTTP transport: `WithCORS` wraps the standard handler without forcing this library to own the application's HTTP server.
- API key use cases are supported through existing `StaticHeaders`, `BeforeRequest`, and opt-in OpenAPI security defaults instead of a single API-key field.

## Intentionally Not Implemented

- Swagger 2.0 parsing. This package targets OpenAPI 3.0 and 3.1 only.
- CLI, stdio transport, and standalone listener management. This package is a Go library that returns `http.Handler` and `*mcp.Server`.
- Non-standard `GET /tools` and `GET /health` endpoints. Tool discovery remains MCP-native; applications can add their own health routes beside the handler.
- Fetching specs from arbitrary URLs. Callers can fetch remote specs with their own HTTP client, policy, and authentication, then pass the reader to `Config.Spec`.

## Existing Coverage That Already Matched

- Runtime conversion from OpenAPI operations to MCP tools.
- JSON/YAML OpenAPI parsing.
- Path, query, and request body argument handling, with additional header and cookie support.
- Base URL inference from the OpenAPI document, with explicit `BaseURL` override.
- HTTP request execution with structured tool results and error signaling for non-2xx responses.
