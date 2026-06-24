.PHONY: test fmt tidy verify example

GOFILES := $(shell find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

fmt:
	gofmt -w $(GOFILES)

tidy:
	go mod tidy

verify: fmt tidy test

example:
	go run ./examples/basic
