# Project settings
PROJECT_NAME := dbp
CMD_DIR := ./src/cmd/dbms/
BIN_DIR := ./bin
BIN := $(BIN_DIR)/$(PROJECT_NAME)

GO := go
GOFLAGS := -trimpath
LDFLAGS := -s -w

# Default target
.PHONY: all
all: build

# Create bin directory
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Build the DBMS binary
.PHONY: build
build: $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN) $(CMD_DIR)

# Run unit tests
.PHONY: test
test:
	$(GO) test ./src/internal/... ./src/pkg/...

# Run all tests (unit + integration)
.PHONY: test-all
test-all:
	$(GO) test ./src/...

# Run tests with race detector
.PHONY: test-race
test-race:
	$(GO) test -race ./src/...

# Format code
.PHONY: fmt
fmt:
	$(GO) fmt ./src/...

# Static analysis
.PHONY: vet
vet:
	$(GO) vet ./src/...

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)

# Quick sanity check before commit
.PHONY: check
check: fmt vet test
