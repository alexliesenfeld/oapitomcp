# Examples

Run `make examples` from the repository root to list the available runnable
examples.

## Basic

Run a small local API and expose it through MCP:

```sh
make example-basic
```

The example mounts:

- `GET /api/pets/{id}` as a sample OpenAPI-backed API endpoint.
- `POST /mcp` as the MCP Streamable HTTP endpoint.

The OpenAPI document is parsed at startup from an in-memory reader; no code is
generated.

## Petstore

Run a local API from a single-file OpenAPI document:

```sh
make example-petstore
```

The example mounts:

- `GET /api/pets` and `POST /api/pets` as sample API endpoints.
- `POST /mcp` as the MCP Streamable HTTP endpoint.

The OpenAPI document is loaded from `petstore/openapi.yaml` with
`NewHandlerFromFile`.

## Multi-file

Run a local API from an OpenAPI document with same-directory refs:

```sh
make example-multifile
```

The example mounts:

- `GET /api/pets/{petId}` as the sample API endpoint.
- `POST /mcp` as the MCP Streamable HTTP endpoint.

The OpenAPI document is loaded from `multifile/openapi.yaml`, which references
`parameters.yaml` and `schemas.yaml`.

## Filtering

Run a local API while exposing only selected operations as MCP tools:

```sh
make example-filtering
```

The backing API includes public, admin, and debug endpoints. `OperationFilter`
exposes only `listUsers` and `createUser` to MCP.

## Documents

- `petstore/openapi.yaml` is a single-file OpenAPI document with parameters, request bodies, responses, and security schemes.
- `multifile/openapi.yaml` references `parameters.yaml` and `schemas.yaml` in the same directory.
- `filtering/openapi.yaml` demonstrates public, admin, and debug endpoints for `OperationFilter` examples.
