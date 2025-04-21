.PHONY: build test clean lint

# Build variables
BINARY_NAME=vectordb
BUILD_DIR=build
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOLINT=golangci-lint

all: test build

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/vectordb

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

lint:
	$(GOLINT) run

# Run the server
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Generate protobuf files
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/*.proto 