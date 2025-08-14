# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the project
make build

# Run the built binary
make run
# or directly:
./build/vectordb

# Run with custom config
./build/vectordb -config path/to/config.yaml
```

### Testing and Quality
```bash
# Run all tests
make test

# Run linting (requires golangci-lint)
make lint

# Clean build artifacts
make clean
```

### Vector Operations via CLI
```bash
# Insert a vector
./build/vectordb -insert "0.1,0.2,0.3" -id "vec1" -metadata "type=test,category=demo"

# Search for similar vectors
./build/vectordb -search "0.1,0.2,0.3" -k 5 -threshold 0.7

# Combined operations work in sequence
```

### Protocol Buffers
```bash
# Generate protobuf files (when proto definitions exist)
make proto
```

## Architecture Overview

### Core Components

**Engine (`internal/engine/`)**
- `engine.go`: Main vector database engine with in-memory storage using `map[string]types.Vector`
- `search.go`: K-nearest neighbor search implementation using brute-force with priority queue
- `errors.go`: Engine-specific error definitions
- Thread-safe operations using `sync.RWMutex`

**Vector Types (`pkg/types/`)**
- `vector.go`: Core data structures for `Vector`, `SearchResult`, and vector math operations
- Supports cosine similarity calculations
- Metadata stored as `map[string]interface{}`

**Configuration (`internal/config/`)**
- YAML-based configuration system
- Supports server, storage, index, database, and logging configuration
- Default config available via `DefaultConfig()`

**Memory Pool (`mempool/`)**
- Cache management system with LRU eviction
- Types and error handling for memory operations
- Currently minimal implementation

**Storage Layer (`storage/`)**
- Simple vector storage implementations
- JSON-based persistence in `data/vectors.json`
- No advanced indexing structures yet (HNSW planned)

### Data Flow
1. Vectors are inserted via CLI or API into the Engine
2. Engine validates dimensions against config
3. Vectors stored in in-memory map and persisted to JSON
4. Search performs brute-force similarity calculation across all vectors
5. Results ranked by cosine similarity with configurable threshold

### Configuration System
- Main config file: `config.yaml`
- Supports runtime configuration of:
  - Server host/port
  - Storage path
  - Index type and dimensions
  - Database limits
  - Logging configuration

### Current Limitations
- No advanced indexing (HNSW implementation planned)
- JSON-based persistence (not production-ready)
- Brute-force search only
- No REST/gRPC API yet (CLI only)
- No distributed features

### Development Status
This is an early-stage vector database implementation. See the comprehensive development checklist in README.md for planned features across 10 modules including persistence, APIs, distribution, and observability.