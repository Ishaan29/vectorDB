# VectorDB

A high-performance cloud native, dictributed vector database implementation in Go.
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

# ✅ VectorDB Development Checklist


---

## 📦 Module 1: Core Engine & In-Memory Store

- [x] Define `Vector` struct (ID, Embedding, Metadata)
- [x] Create `Engine` struct with in-memory store (`map[string]Vector`)
- [x] Add `Insert(Vector)` method to Engine
- [x] Add `Get(ID)` method
- [ ] Add `Update(ID, Vector)` method
- [ ] Add `Delete(ID)` method
- [ ] Define interface `Store` for CRUD
- [ ] Ensure `Engine` satisfies `Store` interface (`var _ Store = (*Engine)(nil)`)

---

## 📌 Module 2: ANN Index (Flat Search)

- [ ] Define `Index` interface: `Insert(id, embedding)`, `Search(query, k)`
- [ ] Implement `FlatIndex` using brute-force search
- [ ] Add cosine similarity function
- [ ] Integrate `FlatIndex` into `Engine`
- [ ] Add vector search method to `Engine`

---

## 💾 Module 3: Persistence Layer (BadgerDB)

- [ ] Add BadgerDB dependency
- [ ] Implement `PersistStore` using BadgerDB
- [ ] Save vector on `Insert`
- [ ] Load vector on `Get`
- [ ] Load all vectors on startup
- [ ] Add snapshotting or WAL abstraction

---

## 🧠 Module 4: Metadata & Hybrid Filtering

- [ ] Support basic metadata filters (e.g., `author == "foo"`)
- [ ] Build simple query engine (e.g., `AND`, `OR`)
- [ ] Allow vector search with metadata filters
- [ ] Index metadata in memory

---

## 🔗 Module 5: Go SDK Interface

- [ ] Create `client` package
- [ ] Add methods: `Insert()`, `Search()`, `Get()`, `Update()`, `Delete()`
- [ ] Support hybrid search with metadata
- [ ] Add batch insert & search methods

---

## 🌐 Module 6: REST API

- [ ] Set up Gin HTTP server
- [ ] `POST /vectors` - Insert vector
- [ ] `GET /vectors/:id` - Get vector
- [ ] `POST /search` - Vector + metadata search
- [ ] `DELETE /vectors/:id` - Delete vector
- [ ] Add basic middleware (logging, validation)

---

## ⚖️ Module 7: Distribution & Replication (Advanced)

- [ ] Design vector sharding strategy
- [ ] Use Raft or etcd for replication
- [ ] Implement follower sync
- [ ] Add vector write propagation
- [ ] Monitor node health / failover

---

## 📊 Module 8: Observability

- [ ] Add `/metrics` Prometheus endpoint
- [ ] Track insert/search latency
- [ ] Track vector count & index size
- [ ] Add structured logging (zap or logrus)

---

## 🚢 Module 9: Deployment

- [ ] Create Dockerfile
- [ ] Add Docker Compose for local dev
- [ ] Add config support via ENV or YAML
- [ ] Add `/healthz` and `/readyz` endpoints
- [ ] Write setup guide & examples in `README.md`

---

## 🔒 Module 10: Security & Access Control (Optional)

- [ ] Add API key or JWT-based auth
- [ ] Define basic user roles (read/write)
- [ ] Enforce access control on API routes
- [ ] Support HTTPS (TLS cert config)

---

## 🧠 Optional Enhancements

- [ ] Embedding compression (e.g., PQ)
- [ ] Versioned vector records
- [ ] Delta sync across nodes
- [ ] Snapshot export (e.g., S3)
- [ ] Integrated vectorizer (e.g., OpenAI API wrapper)

---

## 📍 Project Progress Tracker

| Module | Feature | Status |
|--------|---------|--------|
| 1 | Core Engine + Memory | 🔲 |
| 2 | Flat Index | 🔲 |
| 3 | Persistence (BadgerDB) | 🔲 |
| 4 | Metadata Filtering | 🔲 |
| 5 | Go SDK | 🔲 |
| 6 | REST API | 🔲 |
| 7 | Distributed Mode | 🔲 |
| 8 | Metrics & Logging | 🔲 |
| 9 | Deployment | 🔲 |
| 10 | Security | 🔲 |

---