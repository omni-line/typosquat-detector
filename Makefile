.PHONY: build test lint fmt vet clean corpus

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X github.com/omni-line/typosquat-detector/internal/version.Version=$(VERSION)

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/typosquat-detector ./cmd/typosquat-detector

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed; running go vet only"; go vet ./...; exit 0; }
	golangci-lint run ./...

corpus:
	go run ./scripts/update-corpus -out internal/corpus

clean:
	rm -rf bin dist
