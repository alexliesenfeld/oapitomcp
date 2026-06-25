.PHONY: test fmt tidy verify examples example example-basic example-petstore example-multifile example-filtering

GOFILES := $(shell find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

fmt:
	gofmt -w $(GOFILES)

tidy:
	go mod tidy

verify: fmt tidy test

examples:
	@printf '%s\n' \
		'make example-basic      # in-memory OpenAPI document' \
		'make example-petstore   # single-file OpenAPI document' \
		'make example-multifile  # local multi-file refs' \
		'make example-filtering  # OperationFilter include/exclude behavior'

example: example-basic

example-basic:
	go run ./examples/basic

example-petstore:
	go run ./examples/petstore

example-multifile:
	go run ./examples/multifile

example-filtering:
	go run ./examples/filtering
