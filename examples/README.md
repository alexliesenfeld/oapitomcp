# Examples

## Basic

Run a small local API and expose it through MCP:

```sh
make example
```

The example mounts:

- `GET /api/pets/{id}` as a sample OpenAPI-backed API endpoint.
- `POST /mcp` as the MCP Streamable HTTP endpoint.

The OpenAPI document is parsed at startup from an in-memory reader; no code is
generated.

## Document Fixtures

- `petstore/openapi.yaml` is a single-file OpenAPI document with parameters, request bodies, responses, and security schemes.
- `multifile/openapi.yaml` references `parameters.yaml` and `schemas.yaml` in the same directory.
- `filtering/openapi.yaml` demonstrates public, admin, and debug endpoints for `OperationFilter` examples.
