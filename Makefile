.PHONY: build test test-integration lint compose-up compose-down smoke smoke-registry

# Override for CI (Linux): make test TEST_FLAGS=-race
# On Windows, leave empty; -race requires CGO.
TEST_FLAGS ?=

build:
	go build -o bin/verity ./cmd/verity
	go build -o bin/verity-api ./cmd/verity-api
test:
	go test $(TEST_FLAGS) -coverprofile=coverage.out ./...
test-integration:
	go test -p 1 $(TEST_FLAGS) -tags=integration -coverprofile=coverage-integration.out ./internal/metadata/... ./internal/db/... ./internal/registry/...
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
