.PHONY: build test test-lineage lint smoke-lineage smoke-analyze

# Override for CI (Linux): make test TEST_FLAGS=-race
# On Windows, leave empty; -race requires CGO.
TEST_FLAGS ?=

VERSION ?= dev
LDFLAGS := -X github.com/BrendenWalker/lineagis/internal/version.Version=$(VERSION)
GOEXE := $(shell go env GOEXE)
LINEAGIS_BIN := bin/lineagis$(GOEXE)

build:
	go build -ldflags "$(LDFLAGS)" -o $(LINEAGIS_BIN) ./cmd/lineagis

test:
	go test $(TEST_FLAGS) -coverprofile=coverage.out ./...

test-lineage:
	go test $(TEST_FLAGS) ./internal/core/... ./internal/ingest/... ./internal/analyze/... ./internal/normalize/... ./internal/lineage/... ./internal/storage/memory/... ./tests/conformance/... ./cmd/lineagis/...

lint:
	golangci-lint run ./...

smoke-lineage: build
	bash scripts/smoke-lineage.sh

smoke-analyze: build
	bash scripts/smoke-analyze.sh
