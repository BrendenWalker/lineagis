.PHONY: build test lint compose-up compose-down

# Override for CI (Linux): make test TEST_FLAGS=-race
# On Windows, leave empty; -race requires CGO.
TEST_FLAGS ?=

build:
	go build -o bin/verity ./cmd/verity
test:
	go test $(TEST_FLAGS) -coverprofile=coverage.out ./...
lint:
	golangci-lint run ./...
compose-up:
	docker compose up -d --build --wait
compose-down:
	docker compose down
