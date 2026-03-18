.PHONY: fmt test build eval eval-real

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

test:
	go test ./...

build:
	go build ./...

eval:
	go test ./evals/... -v

eval-real:
	CANX_EVAL_REAL=1 go test ./evals/agentic -run TestAgenticRealExecSmokeIfEnabled -v
