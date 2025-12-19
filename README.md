# dockstart

A CLI tool that analyzes a project and generates Docker development environment files for VS Code Dev Containers.

**Learning project** for Docker (Dockerfile fundamentals + Compose) and Go.

## Features

- **Language Detection**: Automatically detects Node.js, Go, Python, and Rust projects
- **Service Detection**: Identifies PostgreSQL and Redis dependencies
- **Log Aggregation**: Auto-configures Fluent Bit sidecar when structured logging libraries detected
- **Complete Dev Environment**: Generates devcontainer.json, docker-compose.yml, and Dockerfile
- **VS Code Ready**: Generated files work with VS Code's Dev Containers extension

## Installation

```bash
# Build from source
go build -o dockstart ./cmd/dockstart

# Or run with Docker
docker build -t dockstart .
docker run -v $(pwd):/project dockstart /project
```

## Usage

### Basic Usage

```bash
# Generate dev environment for current directory
dockstart .

# Generate for a specific project
dockstart ./my-project
```

### Options

```bash
# Preview output without writing files
dockstart --dry-run ./my-project

# Overwrite existing files
dockstart --force ./my-project
```

## Example Output

### Node.js Project with PostgreSQL

```bash
$ dockstart --dry-run ./express-app

📂 Analyzing ./express-app...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...

✨ Done!
```

Generated files:

**`.devcontainer/devcontainer.json`**
```json
{
  "name": "express-app",
  "dockerComposeFile": "docker-compose.yml",
  "service": "app",
  "workspaceFolder": "/workspace",
  "customizations": {
    "vscode": {
      "extensions": ["dbaeumer.vscode-eslint"]
    }
  },
  "forwardPorts": [3000, 5432],
  "postCreateCommand": "npm install",
  "remoteUser": "node"
}
```

**`.devcontainer/docker-compose.yml`**
```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - ..:/workspace:cached
    depends_on:
      - postgres
    environment:
      - DATABASE_URL=postgres://postgres:postgres@postgres:5432/express-app_dev

  postgres:
    image: postgres:16-alpine
    volumes:
      - postgres-data:/var/lib/postgresql/data
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: express-app_dev

volumes:
  postgres-data:
```

## Supported Languages

| Language | Config File | Version Detection | Default Port |
|----------|------------|-------------------|--------------|
| Node.js | package.json | engines.node | 3000 |
| Go | go.mod | go directive | 8080 |
| Python | pyproject.toml / requirements.txt | requires-python | 8000 |
| Rust | Cargo.toml | rust-version / edition | 8080 |

## Detected Services

| Service | Node.js | Go | Python | Rust |
|---------|---------|----|---------| -----|
| PostgreSQL | pg, prisma, typeorm | pgx, lib/pq | psycopg2, sqlalchemy | sqlx, diesel |
| Redis | redis, ioredis, bull | go-redis | redis, celery | redis |

## Log Aggregator Sidecar

When dockstart detects structured logging libraries in your project, it automatically generates a **Fluent Bit** log aggregator sidecar. This provides centralized logging for your development environment.

### Detected Logging Libraries

| Language | JSON Loggers | Text Loggers |
|----------|-------------|--------------|
| Node.js | pino, bunyan | winston, morgan |
| Go | zap, zerolog | logrus, slog |
| Python | structlog, python-json-logger | loguru, logbook |
| Rust | tracing, slog | log, env_logger |

### Example with Logging

```bash
$ dockstart --dry-run ./my-express-app

📂 Analyzing ./my-express-app...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]
   📋 Logging: [pino] (JSON format)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating fluent-bit.conf...

✨ Done!
```

### Generated Sidecar Configuration

When logging is detected, the following is added to `docker-compose.yml`:

```yaml
services:
  app:
    # ... app configuration ...
    depends_on:
      - fluent-bit
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: app.my-app

  fluent-bit:
    image: fluent/fluent-bit:latest
    volumes:
      - ./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf:ro
    ports:
      - "24224:24224"
```

### Viewing Logs

Logs from your application are collected by Fluent Bit and output to stdout. View them with:

```bash
# In the devcontainer terminal
docker compose logs -f fluent-bit

# Or view all logs
docker compose logs -f
```

See [docs/sidecars/log-aggregator.md](docs/sidecars/log-aggregator.md) for detailed documentation.

## Generated Files

### devcontainer.json
- VS Code Dev Container configuration
- Language-specific base image or docker-compose reference
- VS Code extensions for the language
- Port forwarding
- Post-create commands

### docker-compose.yml (when services or logging detected)
- App service with build context
- PostgreSQL service with named volume
- Redis service with named volume
- Fluent Bit log aggregator sidecar (when logging libraries detected)
- Environment variables for service URLs

### Dockerfile
- Language-specific base image
- Common dev tools (git, curl, wget, vim)
- WORKDIR set to /workspace

## Development

```bash
# Build
go build -o dockstart ./cmd/dockstart

# Run tests
go test ./...

# Build Docker image
docker build -t dockstart .
```

## Project Structure

```
dockstart/
├── cmd/dockstart/          # CLI entry point
│   └── cmd/                # Cobra commands
├── docs/
│   ├── adr/                # Architecture Decision Records
│   ├── sidecars/           # Sidecar documentation
│   └── examples/           # Example projects
├── internal/
│   ├── detector/           # Language detection
│   │   ├── node.go        # Node.js detector
│   │   ├── golang.go      # Go detector
│   │   ├── python.go      # Python detector
│   │   └── rust.go        # Rust detector
│   ├── generator/          # File generation
│   │   ├── devcontainer.go
│   │   ├── compose.go
│   │   ├── dockerfile.go
│   │   ├── logsidecar.go  # Fluent Bit generator
│   │   └── templates/
│   └── models/             # Data structures
└── Dockerfile              # Multi-stage container build
```

## License

MIT
