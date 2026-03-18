.PHONY: fmt test build eval eval-real report report-real

fmt:
	gofmt -w $(shell find . -type f -name '*.go' -not -path './vendor/*')

test:
	go test ./...

build:
	go build ./...

eval:
	go test ./evals/... -v

eval-real:
	CANX_EVAL_REAL=1 go test ./evals/agentic -run 'TestAgenticRealExecSmokeIfEnabled|TestPlannerRealSmokeIfEnabled' -v

report:
	go run ./cmd/canx-eval-report

report-real:
	go run ./cmd/canx-eval-report -real
