SHELL := /bin/sh

GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: build guard-go guard-go-module guard-golangci-lint lint test test-unit test-e2e verify

guard-go:
	@command -v $(GO) >/dev/null 2>&1 || { \
		echo "error: '$(GO)' not found in PATH"; \
		exit 1; \
	}

guard-go-module: guard-go
	@test -f go.mod || { \
		echo "error: go.mod not found."; \
		echo "hint: complete T02 in docs/plans/2026-04-10-go-symphony-design-task.md before running Go targets."; \
		exit 1; \
	}

guard-golangci-lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || { \
		echo "error: '$(GOLANGCI_LINT)' not found in PATH"; \
		exit 1; \
	}

build: guard-go-module
	$(GO) build ./...

lint: guard-go-module guard-golangci-lint
	$(GOLANGCI_LINT) run ./...

test: guard-go-module
	$(GO) test ./...

test-unit: guard-go-module
	$(GO) test -short ./...

test-e2e: guard-go-module
	$(GO) test -count=1 -tags=e2e ./...

verify: build lint test test-e2e
