# Higress Feature Review

Reviewed project: <https://github.com/higress-group/openapi-to-mcpserver>

## Implemented In This Library

- OpenAPI paths become MCP tools at runtime.
- JSON and YAML OpenAPI documents are supported through kin-openapi.
- Tool names can be prefixed with `Config.ToolNamePrefix`.
- Path, query, header, cookie, JSON body, and form body inputs are supported.
- Tool descriptions preserve OpenAPI summaries, descriptions, tags, security names, response codes, and response field documentation.
- Response-derived MCP output schemas are generated while keeping the library response wrapper stable.
- Structured MCP responses include status, content type, and parsed body.
- OpenAPI security schemes are exposed on `Catalog.SecuritySchemes`.
- `x-defaultCredential` and legacy `defaultCredential` are recognized, with opt-in runtime application through `Config.UseSecurityDefaults`.
- Static headers can be applied to every upstream request through `Config.StaticHeaders`.

## Intentionally Not Implemented

- Higress YAML config generation. This package serves a live MCP HTTP handler instead of producing gateway config.
- File-output formats and CLI flags. The public API is Go-first.
- YAML template patch files. Runtime customization is provided through config fields and `BeforeRequest`.
