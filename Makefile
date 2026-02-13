.PHONY: all build clean test fmt vet install generator sender

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=gofmt
GOVET=$(GOCMD) vet

# Binary names
GENERATOR_BINARY=telemetry-generator
SENDER_BINARY=telemetry-sender

# Build directory
BUILD_DIR=./build

all: fmt vet test build

build: generator sender

generator:
	@echo "Building $(GENERATOR_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(GENERATOR_BINARY) ./cmd/telemetry-generator

sender:
	@echo "Building $(SENDER_BINARY)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(SENDER_BINARY) ./cmd/telemetry-sender

test:
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

install: build
	@echo "Installing binaries..."
	@cp $(BUILD_DIR)/$(GENERATOR_BINARY) $(GOPATH)/bin/
	@cp $(BUILD_DIR)/$(SENDER_BINARY) $(GOPATH)/bin/

deps:
	@echo "Downloading dependencies..."
	$(GOCMD) mod download
	$(GOCMD) mod tidy

help:
	@echo "Available targets:"
	@echo "  all         - Format, vet, test, and build"
	@echo "  build       - Build both generator and sender"
	@echo "  generator   - Build telemetry-generator"
	@echo "  sender      - Build telemetry-sender"
	@echo "  test        - Run tests"
	@echo "  fmt         - Format code"
	@echo "  vet         - Run go vet"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install binaries to GOPATH/bin"
	@echo "  deps        - Download and tidy dependencies"
