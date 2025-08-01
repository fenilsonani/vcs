.PHONY: build test bench lint clean run

# Build variables
BINARY_NAME=vcs
BUILD_DIR=./build
MAIN_PACKAGE=./cmd/vcs

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Build flags for optimization
LDFLAGS=-ldflags "-w -s"
GCFLAGS=-gcflags="all=-N -l"

# Default target
all: test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build with debug symbols
build-debug:
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GCFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Debug build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./benchmarks/...

# Quick benchmark - Hyperdrive performance tests
bench-quick:
	@echo "Running quick performance benchmarks..."
	$(GOTEST) -run=^$$ -bench=BenchmarkHyperdrive ./cmd/vcs -benchtime=1s

# Full performance suite
bench-full:
	@echo "Running full benchmark suite..."
	$(GOTEST) -run=^$$ -bench=. ./cmd/vcs -benchtime=10s -benchmem

# Hardware acceleration tests
bench-hardware:
	@echo "Running hardware acceleration benchmarks..."
	$(GOTEST) -run=^$$ -bench="BenchmarkARM64|BenchmarkAssembly|BenchmarkFPGA" ./cmd/vcs -benchtime=5s

# Large repository simulation
bench-large-repos:
	@echo "Running large repository benchmarks..."
	$(GOTEST) -run=^$$ -bench=BenchmarkLargeRepositories ./cmd/vcs -benchtime=30s -timeout=10m

# Compare with Git
bench-compare:
	@echo "Running Git comparison benchmarks..."
	$(GOTEST) -run=^$$ -bench=BenchmarkGitComparison ./cmd/vcs -benchtime=10s

# Memory allocator benchmarks
bench-memory:
	@echo "Running memory allocator benchmarks..."
	$(GOTEST) -run=^$$ -bench="BenchmarkMemory|BenchmarkAllocator" ./cmd/vcs -benchtime=5s -benchmem

# Extreme concurrency benchmarks
bench-concurrent:
	@echo "Running concurrency benchmarks..."
	$(GOTEST) -run=^$$ -bench="BenchmarkExtremeConcurrency|BenchmarkScalability" ./cmd/vcs -benchtime=10s

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Vet code
vet:
	@echo "Vetting code..."
	$(GOVET) ./...

# Lint code (requires golangci-lint)
lint:
	@echo "Linting code..."
	golangci-lint run ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

# Install the binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Development mode with hot reload (requires air)
dev:
	air -c .air.toml

# Performance profiling
profile:
	@echo "Running CPU profile..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./benchmarks
	$(GOCMD) tool pprof cpu.prof

# Memory profiling
memprofile:
	@echo "Running memory profile..."
	$(GOTEST) -memprofile=mem.prof -bench=. ./benchmarks
	$(GOCMD) tool pprof mem.prof