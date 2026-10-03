.PHONY: build test race lint fmt vet fuzz vuln clean corpus

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
PKG     := github.com/omni-line/typosquat-detector/internal/version
LDFLAGS := -s -w -X $(PKG).Version=$(VERSION) -X $(PKG).Commit=$(COMMIT) -X $(PKG).Date=$(DATE)
FUZZTIME ?= 30s

build:
	@mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/typosquat-detector ./cmd/typosquat-detector

test:
	go test ./...

race:
	go test -race ./...

vet:
	go vet ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed; running go vet only"; go vet ./...; exit 0; }
	golangci-lint run ./...

# Fuzz every parser of untrusted input plus the distance kernel.
fuzz:
	go test ./internal/distance -run='^$$' -fuzz=FuzzWithin -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/npm -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/pypi -run='^$$' -fuzz=FuzzParseRequirements -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/pypi -run='^$$' -fuzz=FuzzParsePyProject -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/composer -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/gomod -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/cargo -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/maven -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/rubygems -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/docker -run='^$$' -fuzz=FuzzParseDockerfile -fuzztime=$(FUZZTIME)
	go test ./internal/manifest/conan -run='^$$' -fuzz=FuzzParse -fuzztime=$(FUZZTIME)

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

corpus:
	go run ./scripts/update-corpus -out internal/corpus

clean:
	rm -rf bin dist
