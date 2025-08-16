# VectorDB

A high-performance cloud native, distributed vector database implementation in Go.
With hyper focus in text and code embeddings. 

## Features

- Fast vector similarity search
- Support for multiple index types (HNSW planned)
- Metadata storage and filtering
- REST and gRPC APIs (coming soon)
- Persistence and recovery
- Production-ready configuration

## Project Structure

```
vectorDB/
├── api/                    # API layer (gRPC/HTTP handlers)
├── cmd/                    # Main applications
├── internal/              # Private application code
├── pkg/                   # Public libraries
├── proto/                 # Protocol buffer definitions
├── scripts/              # Build and maintenance scripts
├── test/                 # Additional test files
└── Makefile             # Build automation
```
# v0.1 Architecture
  * BadgerDB: Persistent storage for full vectors with metadata
  * HNSW Index: Fast in-memory similarity search
  * Engine: Orchestrates both components with proper error handling
  * Dual-write pattern: Ensures durability before searchability
  * Startup recovery: Rebuilds index from persisted data
  * Thread-safe operations: Proper mutex usage throughout

# e2e Flow
  Insert: BadgerDB → HNSW
  Search: HNSW → BadgerDB (hydration)
  Startup: BadgerDB → HNSW (rebuild)

## Getting Started

### Prerequisites

- Go 1.21 or later
- Make

### Building

```bash
make build
```

### Running

```bash
# Using default config
./build/vectordb

# Using custom config
./build/vectordb -config path/to/config.yaml
```

## Configuration

Configuration is handled through a YAML file. Here's an example configuration:

```yaml
server:
  host: localhost
  port: 8080

storage:
  path: data

index:
  type: hnsw
  dimensions: 128

database:
  max_vectors: 1000000
```

## Development
```bash
make build
go run cmd/vectordb/main.go -config config.yaml
```
### Running Tests

```bash
make test
```

### Linting

```bash
make lint
```

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
