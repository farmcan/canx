.PHONY: fmt test build

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

test:
	go test ./...

build:
	go build ./...
