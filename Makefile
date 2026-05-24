.PHONY: build test lint
build:
	go build -o bin/verity ./cmd/verity
test:
	go test -race -coverprofile=coverage.out ./...
lint:
	golangci-lint run ./...
