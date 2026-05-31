.PHONY: build test test-integration lint compose-up compose-down smoke smoke-registry operator-stack-ci

# Override for CI (Linux): make test TEST_FLAGS=-race
# On Windows, leave empty; -race requires CGO.
TEST_FLAGS ?=

# Release version (default dev). Set VERSION=v0.2.0 for release builds.
VERSION ?= dev
LDFLAGS := -X github.com/BrendenWalker/lineagis/internal/version.Version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/lineagis ./cmd/lineagis
	go build -ldflags "$(LDFLAGS)" -o bin/lineagis-api ./cmd/lineagis-api
test:
	go test $(TEST_FLAGS) -coverprofile=coverage.out ./...
test-integration:
	go test -p 1 $(TEST_FLAGS) -tags=integration -coverprofile=coverage-integration.out ./internal/metadata/... ./internal/db/... ./internal/registry/... ./internal/api/...
lint:
	golangci-lint run ./...
compose-up:
	docker compose up -d --build --wait
compose-down:
	docker compose down
smoke-registry:
	bash scripts/smoke-registry.sh
smoke: compose-up
	bash scripts/smoke-stack.sh
operator-stack-ci:
	bash scripts/operator-stack-ci.sh
