BINARY     := acptui
MODULE     := github.com/lyarwood/acptui
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X '$(MODULE)/internal/cmd.Version=$(VERSION)'"
GO         := go
GINKGO     := ginkgo

.PHONY: all build test test-verbose test-cover lint fmt vet clean install run demo

all: build

build:
	$(GO) build $(LDFLAGS) -o bin/$(BINARY) ./cmd/acptui

test:
	$(GINKGO) run -r ./internal/...

test-verbose:
	$(GINKGO) run -r -v ./internal/...

test-cover:
	$(GINKGO) run -r --coverprofile=coverage.out ./internal/...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint:
	golangci-lint run ./internal/... ./cmd/...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf bin/ coverage.out coverage.html

install: build
	install -m 0755 bin/$(BINARY) $(shell $(GO) env GOPATH)/bin/$(BINARY)

run: build
	./bin/$(BINARY)

demo: build
	@mkdir -p /tmp/acptui-demo
	@cp demo/config.json /tmp/acptui-demo/
	@echo "Starting mock API server..."
	@$(GO) run demo/mock-server.go & echo $$! > /tmp/acptui-demo/server.pid
	@sleep 1
	vhs demo.tape
	@kill $$(cat /tmp/acptui-demo/server.pid) 2>/dev/null || true
	@rm -rf /tmp/acptui-demo
