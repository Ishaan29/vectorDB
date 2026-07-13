.PHONY: all build build-nocgo build-server test test-nocgo test-all bench bench-nocgo bench-e2e bench-e2e-quick clean lint run proto docker-verify

# Build variables
BINARY_NAME=vectordb
BUILD_DIR=build

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOLINT=golangci-lint

# CGO is stated explicitly on every target so behavior does not depend on the
# developer's environment:
#   CGO_ENABLED=1 -> C++ SIMD math core (pkg/vectormath/simd); needs a C++
#                    toolchain (clang on macOS, g++ on Linux).
#   CGO_ENABLED=0 -> pure-Go fallback (pkg/vectormath/scalar); builds anywhere.

all: test-all build

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vectordb

build-nocgo:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-nocgo ./cmd/vectordb

build-server:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-server ./cmd/vectordb-server

test:
	CGO_ENABLED=1 $(GOTEST) -v ./...

test-nocgo:
	CGO_ENABLED=0 $(GOTEST) -v ./...

test-all: test test-nocgo

bench:
	CGO_ENABLED=1 $(GOTEST) -bench=. -benchmem -run='^$$' -count=6 ./pkg/vectormath/... ./internal/index/...

# End-to-end benchmark harness (100k vectors by default): build/ingest
# throughput, latency percentiles, recall@k, concurrent QPS, memory.
# Results land in bench/results/. See bench/README.md for flags.
bench-e2e:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-bench ./cmd/bench
	./$(BUILD_DIR)/$(BINARY_NAME)-bench

# Quick smoke version of the e2e bench (~30s)
bench-e2e-quick:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-bench ./cmd/bench
	./$(BUILD_DIR)/$(BINARY_NAME)-bench -n 10000 -queries 200 -concurrent-ops 1000 -label quick

bench-nocgo:
	CGO_ENABLED=0 $(GOTEST) -bench=. -benchmem -run='^$$' -count=6 ./pkg/vectormath/... ./internal/index/...

# Verify the cgo SIMD core builds and passes tests on Linux (both arches).
# linux/arm64 runs at native speed on Apple Silicon and exercises the NEON
# path under Linux; linux/amd64 runs under QEMU (slow) and exercises the
# portable scalar C++ path.
docker-verify:
	docker build -f Dockerfile.verify --platform linux/arm64 -t vectordb-verify:arm64 .
	docker build -f Dockerfile.verify --platform linux/amd64 -t vectordb-verify:amd64 .

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# Note: golangci-lint needs CGO_ENABLED=1 to typecheck the cgo bindings.
lint:
	CGO_ENABLED=1 $(GOLINT) run

# Run the server
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Generate protobuf files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto
