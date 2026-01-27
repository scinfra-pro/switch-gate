.PHONY: help build build-linux build-all run test test-cover lint fmt tidy clean docker-build docker-run version

BINARY := switch-gate
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
.DEFAULT_GOAL := help

# === Help ===

help: ## Show this help
	@echo ""
	@echo "  switch-gate $(VERSION)"
	@echo "  ========================"
	@echo ""
	@echo "  Usage: make [target]"
	@echo ""
	@echo "  Build:"
	@echo "    build          Build for current OS"
	@echo "    build-linux    Build for Linux (amd64)"
	@echo "    build-all      Build for all platforms"
	@echo "    clean          Remove build artifacts"
	@echo ""
	@echo "  Development:"
	@echo "    run            Build and run locally"
	@echo "    test           Run tests"
	@echo "    test-cover     Run tests with coverage"
	@echo "    lint           Run linter"
	@echo "    fmt            Format code"
	@echo "    tidy           Tidy go.mod"
	@echo ""
	@echo "  Docker:"
	@echo "    docker-build   Build Docker image"
	@echo "    docker-run     Run in Docker"
	@echo ""
	@echo "  Info:"
	@echo "    version        Show version"
	@echo ""

# === Build ===

build: ## Build for current OS
	@echo "Building switch-gate $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/switch-gate
	@echo "Output: bin/$(BINARY)"

build-linux: ## Build for Linux (amd64)
	@echo "Building switch-gate $(VERSION) for Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/switch-gate
	@echo "Output: bin/$(BINARY)-linux-amd64"

build-all: ## Build for all platforms
	@echo "Building switch-gate $(VERSION) for all platforms..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-amd64 ./cmd/switch-gate
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-linux-arm64 ./cmd/switch-gate
	@echo "Output: bin/$(BINARY)-linux-amd64, bin/$(BINARY)-linux-arm64"

# === Development ===

run: build ## Build and run locally
	@echo "Starting switch-gate..."
	@./bin/$(BINARY) -config configs/switch-gate.example.yaml

test: ## Run tests
	@echo "Running tests..."
	@go test -v -race ./...

test-cover: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...

tidy: ## Tidy go.mod
	@echo "Tidying go.mod..."
	@go mod tidy

# === Clean ===

clean: ## Remove build artifacts
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# === Docker ===

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t switch-gate:$(VERSION) .
	@echo "Image: switch-gate:$(VERSION)"

docker-run: ## Run in Docker
	@echo "Running in Docker..."
	@docker run -p 18388:18388 -p 9090:9090 \
		-v $(PWD)/configs:/etc/switch-gate \
		switch-gate:$(VERSION)

# === Info ===

version: ## Show version
	@echo "switch-gate $(VERSION)"
