# Dockstart Design Document

**Date:** 2025-12-15
**Status:** Approved
**Goal:** Learn Docker (Dockerfile fundamentals + Compose) through building a practical CLI tool in Go

---

## Overview

**dockstart** is a CLI tool that analyzes a project directory and generates Docker development environment files.

```
$ dockstart ./my-project

Analyzing my-project...
   Detected: Node.js (package.json)
   Detected: PostgreSQL (prisma/schema.prisma)
   Detected: Redis (docker-compose.yml reference)

Generated:
   .devcontainer/devcontainer.json
   .devcontainer/docker-compose.yml
   .devcontainer/Dockerfile
```

### Learning Goals

| Go Concepts | Docker Concepts |
|-------------|-----------------|
| CLI with cobra/flags | Dockerfile multi-stage builds |
| File system walking | Docker Compose services |
| JSON marshaling | DevContainer spec |
| Structs & interfaces | Base images & layers |
| Error handling | Volume mounts |
| Testing in Go | Container networking |

### Scope (v1 Minimal)

- **Detect languages:** Node.js, Python, Go, Rust
- **Detect services:** PostgreSQL, Redis
- **Generate:** `devcontainer.json`, `docker-compose.yml`, `Dockerfile`

---

## Architecture

### Project Layout

```
dockstart/
├── cmd/
│   └── dockstart/
│       └── main.go           # CLI entry point
├── internal/
│   ├── detector/
│   │   ├── detector.go       # Interface + orchestrator
│   │   ├── node.go           # Node.js detection
│   │   ├── python.go         # Python detection
│   │   ├── golang.go         # Go detection
│   │   └── rust.go           # Rust detection
│   ├── generator/
│   │   ├── devcontainer.go   # Generate devcontainer.json
│   │   ├── compose.go        # Generate docker-compose.yml
│   │   └── dockerfile.go     # Generate Dockerfile
│   └── models/
│       └── project.go        # Shared data structures
├── templates/                 # Embedded template files
│   ├── devcontainer.json.tmpl
│   ├── docker-compose.yml.tmpl
│   └── Dockerfile.tmpl
├── Dockerfile                 # dockstart's own container
├── go.mod
└── README.md
```

### Data Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Detect    │ ──→ │   Project   │ ──→ │  Generate   │
│  (scan dir) │     │   (model)   │     │  (output)   │
└─────────────┘     └─────────────┘     └─────────────┘
     │                    │                    │
  Reads:              Contains:            Writes:
  - package.json      - Language           - devcontainer.json
  - go.mod            - Version            - docker-compose.yml
  - pyproject.toml    - Services[]         - Dockerfile
  - Cargo.toml        - Dependencies
```

---

## Detection Logic

### Detector Interface

```go
type Detector interface {
    Name() string
    Detect(path string) (*Detection, error)
}

type Detection struct {
    Language    string   // "node", "python", "go", "rust"
    Version     string   // "20", "3.11", "1.21", "1.75"
    Services    []string // ["postgres", "redis"]
    Confidence  float64  // 0.0 - 1.0
}
```

### Detection Rules

| Language | Primary File | Version Source | Service Hints |
|----------|--------------|----------------|---------------|
| Node.js | `package.json` | `engines.node` | `pg`, `redis`, `prisma` |
| Python | `pyproject.toml` / `requirements.txt` | `python_requires` | `psycopg2`, `redis` |
| Go | `go.mod` | `go X.XX` line | `pgx`, `go-redis` |
| Rust | `Cargo.toml` | `rust-version` | `sqlx`, `redis` |

---

## Generated Output

### devcontainer.json

```json
{
  "name": "my-project",
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "workspaceFolder": "/workspace",
  "customizations": {
    "vscode": {
      "extensions": ["golang.go"]
    }
  },
  "forwardPorts": [3000, 5432, 6379],
  "postCreateCommand": "go mod download"
}
```

### docker-compose.yml

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - ..:/workspace:cached
    command: sleep infinity
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:16
    environment:
      POSTGRES_PASSWORD: postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine

volumes:
  postgres-data:
```

### Dockerfile

```dockerfile
FROM golang:1.21

RUN apt-get update && apt-get install -y \
    git \
    curl \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace
```

---

## Dockstart's Own Dockerfile

Multi-stage build demonstrating Dockerfile best practices:

```dockerfile
# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /src

# Layer caching: deps first
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /dockstart ./cmd/dockstart

# Stage 2: Runtime
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /dockstart /usr/local/bin/dockstart

# Non-root user
RUN adduser -D -u 1000 dockstart
USER dockstart

ENTRYPOINT ["dockstart"]
```

**Concepts demonstrated:**
- Multi-stage builds
- Layer caching optimization
- Static binary compilation
- Minimal runtime image (~15MB)
- Security (non-root user)

---

## Implementation Phases

### Phase 1: Skeleton
- Create repo, `go mod init`
- Basic CLI with Cobra
- Scaffold project structure
- Write dockstart's own Dockerfile
- Verify `docker build` works

### Phase 2: Detection
- Implement Detector interface
- Node.js detector
- Go detector
- Unit tests for detectors

### Phase 3: Generation
- Create templates
- Template rendering with `text/template`
- Write files to `.devcontainer/`
- Test on real project

### Phase 4: Polish
- Add Python + Rust detectors
- Service detection (Postgres, Redis)
- Docker Compose generation
- README with examples

---

## Success Criteria

- [ ] Can run `dockstart ./some-project` and get working devcontainer files
- [ ] dockstart itself builds and runs in Docker
- [ ] Understands Dockerfile multi-stage builds
- [ ] Understands Docker Compose service networking
- [ ] Clean, idiomatic Go code with tests
