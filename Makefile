.PHONY: build test clean server provider install-deps fmt vet lint acceptance-test help

# Variables
BINARY_NAME_SERVER=nahcloud

VERSION?=dev
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Default target
all: build

## Help
help: ## Show this help message
	@echo 'Management commands for NahCloud:'
	@echo
	@echo 'Usage:'
	@echo '  make [target]'
	@echo
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) printf "  %-20s%s\n", $$1, $$2 \
	}' $(MAKEFILE_LIST)

## Dependencies
install-deps: ## Install Go dependencies
	go mod download
	go mod tidy

## Building
build: server ## Build server binary

server: ## Build the NahCloud server
	go build $(LDFLAGS) -o bin/$(BINARY_NAME_SERVER) ./cmd/server



## Internal targets
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

dev-css:
	cd web && npm run watch

run-server:
	go run ./cmd/server

## Testing
test: ## Run unit tests
	go test -v ./...

test-coverage: ## Run tests with coverage
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-all: test ## Run all tests

## Code quality
fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: install-golangci-lint ## Run golangci-lint
	golangci-lint run

install-golangci-lint:
	@which golangci-lint > /dev/null || \
		(echo "Installing golangci-lint..." && \
		 curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin)

## Database
clean-db: ## Remove the SQLite database file
	rm -f nah.db nah.db-shm nah.db-wal

reset-db: clean-db ## Reset the database (clean and restart server to recreate)

## Docker
docker-build: ## Build Docker image
	docker build -t nahcloud:$(VERSION) .

docker-run: ## Run Docker container
	docker run -p 8080:8080 nahcloud:$(VERSION)



## Benchmarking
benchmark: ## Run benchmark tests
	go test -bench=. -benchmem ./...

load-test: ## Run basic load test (requires hey and LOAD_TEST_URL)
	@which hey > /dev/null || (echo "Please install hey: go install github.com/rakyll/hey@latest" && exit 1)
	@test -n "$(LOAD_TEST_URL)" || (echo "Set LOAD_TEST_URL to the endpoint you want to hit" && exit 1)
	hey -n 1000 -c 10 $(LOAD_TEST_URL)

## Documentation
docs: ## Generate documentation (placeholder)
	@echo "API documentation generation not implemented yet"

## Cleanup
clean: ## Clean build artifacts and temporary files
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f nah.db nah.db-shm nah.db-wal


## Release preparation
release-prep: clean test-all lint docs ## Prepare for release (run all checks)

## Internal shortcuts
dev-setup: install-deps
	cd web && npm install
	cd web && npm run build

quick-test: fmt vet test
