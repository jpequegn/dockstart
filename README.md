# dockstart

A CLI tool that analyzes a project and generates Docker development environment files for VS Code Dev Containers.

**Learning project** for Docker (Dockerfile fundamentals + Compose) and Go.

## Features

- **Language Detection**: Automatically detects Node.js, Go, Python, and Rust projects
- **Service Detection**: Identifies PostgreSQL and Redis dependencies
- **Log Aggregation**: Auto-configures Fluent Bit sidecar when structured logging libraries detected
- **Background Workers**: Auto-generates worker sidecars when queue libraries detected (Bull, Celery, etc.)
- **Metrics Stack**: Auto-generates Prometheus + Grafana when metrics libraries detected (prom-client, prometheus-client, etc.)
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

## Background Worker Sidecar

When dockstart detects background job processing frameworks (Bull, Celery, etc.), it automatically generates a **worker sidecar** container using the same-image-different-command pattern.

### How It Works

The worker uses the **same Docker image** as your app but runs with a different command to process background jobs:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  App Container  │     │ Worker Container│     │     Redis       │
│   (npm start)   │────▶│(npm run worker) │◀───▶│  (message bus)  │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │  Same image           │  Same image
        │  Different command    │  Different command
        └───────────────────────┘
```

### Detected Worker Frameworks

| Language | Frameworks | Auto-adds Redis |
|----------|------------|-----------------|
| Node.js | bull, bullmq, bee-queue, agenda | Yes |
| Python | celery, rq, dramatiq, huey, arq | rq, arq only |
| Go | asynq, machinery, gocraft-work | asynq only |
| Rust | sidekiq, apalis, faktory | sidekiq only |

### Example with Worker

```bash
$ dockstart --dry-run ./my-node-api

📂 Analyzing ./my-node-api...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [redis]
   👷 Worker: bullmq (command: npm run worker)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...

✨ Done!
```

### Generated Worker Configuration

When a worker framework is detected:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - ..:/workspace:cached
    depends_on:
      - redis

  worker:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - ..:/workspace:cached
    command: npm run worker
    depends_on:
      - app
      - redis
    environment:
      - REDIS_URL=redis://redis:6379
      - WORKER_CONCURRENCY=2
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data
```

### Scaling Workers

```bash
# Run 3 worker instances
docker compose up -d --scale worker=3

# View worker logs
docker compose logs -f worker

# Restart workers only
docker compose restart worker
```

See [docs/sidecars/background-worker.md](docs/sidecars/background-worker.md) for detailed documentation.

## Database Backup Sidecar

When dockstart detects a database service (PostgreSQL, MySQL, or Redis), it automatically generates a **backup sidecar** that creates scheduled backups of your development database.

### How It Works

The backup sidecar runs as a separate container that:
- Creates automated backups on a schedule (default: daily at 3 AM)
- Rotates old backups automatically (default: 7-day retention)
- Uses the appropriate backup tool for each database type
- Stores backups in the `.devcontainer/backups/` directory

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     App     │     │  Database   │     │  db-backup  │
│             │────▶│  (postgres, │◀────│  (alpine +  │
│             │     │   mysql,    │     │ supercronic)│
└─────────────┘     │   redis)    │     └──────┬──────┘
                    └─────────────┘            │
                                               ▼
                                    ┌─────────────────┐
                                    │   ./backups/    │
                                    └─────────────────┘
```

### Supported Databases

| Database | Backup Tool | Hot Backup |
|----------|-------------|------------|
| PostgreSQL | pg_dump | Yes |
| MySQL | mysqldump | Yes |
| Redis | redis-cli + docker cp | Yes |

### Example with Backup

```bash
$ dockstart --dry-run ./my-api

📂 Analyzing ./my-api...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]
   💾 Backup: enabled (daily at 3 AM)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.backup...
📝 Generating backup scripts...

✨ Done!
```

### Generated Backup Configuration

When a database is detected:

```yaml
services:
  app:
    # ... app configuration ...

  postgres:
    image: postgres:16-alpine
    volumes:
      - postgres-data:/var/lib/postgresql/data

  db-backup:
    build:
      context: .
      dockerfile: Dockerfile.backup
    volumes:
      - ./backups:/backup
    depends_on:
      - postgres
    environment:
      - RETENTION_DAYS=7
      - DB_HOST=postgres
      - DB_NAME=myapp_dev
    restart: unless-stopped

volumes:
  postgres-data:
  backups:
```

### Managing Backups

```bash
# View backup files
ls -la .devcontainer/backups/

# Trigger manual backup
docker compose exec db-backup /usr/local/bin/backup.sh

# View backup logs
docker compose logs -f db-backup

# Restore from backup (PostgreSQL)
gunzip -c .devcontainer/backups/postgres-2025-12-20T03-00-00.sql.gz | \
  docker compose exec -T postgres psql -U postgres -d myapp_dev
```

See [docs/sidecars/backup.md](docs/sidecars/backup.md) for detailed documentation.

## File Processing Sidecar

When dockstart detects file upload libraries (multer, python-multipart, etc.), it automatically generates a **file processor sidecar** that watches for uploaded files and processes them (resize images, extract text from PDFs, generate thumbnails).

### How It Works

The file processor sidecar monitors a shared volume for new uploads and processes them automatically:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────────┐
│     App     │     │   uploads   │     │  file-processor │
│             │────▶│  (volume)   │◀────│  (ImageMagick,  │
│ saves files │     │             │     │   Poppler,      │
└─────────────┘     └─────────────┘     │   FFmpeg)       │
       │                   │            └────────┬────────┘
       ▼                   ▼                     │
  /uploads/pending    /uploads/processed         │
                                                 ▼
                                       /uploads/processed
```

### Detected Upload Libraries

| Language | Libraries |
|----------|-----------|
| Node.js | multer, formidable, busboy, express-fileupload |
| Python | python-multipart, aiofiles, flask-uploads |
| Go | multipart (standard library) |
| Rust | actix-multipart, multer |

### Example with File Upload

```bash
$ dockstart --dry-run ./my-upload-app

📂 Analyzing ./my-upload-app...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]
   📁 Uploads: multer detected
   🔗 Sidecars: [file-processor]

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.processor...
📝 Generating processing scripts...

✨ Done!
```

### Generated File Processor Configuration

When upload libraries are detected:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - uploads:/uploads
    environment:
      - UPLOAD_PATH=/uploads/pending
      - PROCESSED_PATH=/uploads/processed
      - FAILED_PATH=/uploads/failed

  file-processor:
    build:
      context: .
      dockerfile: Dockerfile.processor
    volumes:
      - uploads:/uploads
    depends_on:
      - app
    environment:
      - PENDING_PATH=/uploads/pending
      - PROCESSED_PATH=/uploads/processed
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
    restart: unless-stopped

volumes:
  uploads:
```

### Upload Directory Structure

```
/uploads/
├── pending/      # App writes uploaded files here
├── processing/   # Files being processed (temporary)
├── processed/    # Successfully processed files
└── failed/       # Files that failed processing
```

### Using File Uploads

1. Configure your app to save uploads to `/uploads/pending`
2. The processor sidecar automatically detects new files
3. Files are processed based on type (images resized, PDFs text-extracted)
4. Processed files appear in `/uploads/processed`
5. Your app reads from `/uploads/processed`

```javascript
// Express/multer example
const upload = multer({
  dest: process.env.UPLOAD_PATH || '/uploads/pending'
});

app.post('/upload', upload.single('image'), (req, res) => {
  // File is saved to /uploads/pending
  // Processor will move it to /uploads/processed
  const processedPath = req.file.path.replace('pending', 'processed');
  res.json({ processed: processedPath });
});
```

### Processing Capabilities

| File Type | Processing | Output |
|-----------|-----------|--------|
| Images (jpg, png, gif, webp) | Resize, thumbnails, optimize | Original + thumbnail |
| PDFs | Text extraction, first page thumbnail | Text file + thumbnail |
| Videos (mp4, webm, mov) | Thumbnail, metadata, GIF preview | Thumbnail + info.json |

See [docs/sidecars/file-processor.md](docs/sidecars/file-processor.md) for detailed documentation.

## Metrics Stack Sidecar (Prometheus + Grafana)

When dockstart detects Prometheus client libraries (prom-client, prometheus/client_golang, etc.), it generates a complete metrics observability stack.

### How It Works

The metrics stack provides automatic monitoring for your application:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     App     │     │ Prometheus  │     │   Grafana   │
│  /metrics   │────▶│  (scrapes)  │────▶│ (dashboards)│
└─────────────┘     └─────────────┘     └─────────────┘
       ▲                   │                   │
       │              every 15s               │
       └──────────────────────────────────────┘
                    pre-built dashboards
```

### Detected Metrics Libraries

| Language | Libraries |
|----------|-----------|
| Node.js | prom-client, express-prometheus-middleware |
| Go | prometheus/client_golang, prometheus/promhttp |
| Python | prometheus-client, prometheus-fastapi-instrumentator |
| Rust | prometheus, metrics |

### Example with Metrics

```bash
$ dockstart --dry-run ./my-api

📂 Analyzing ./my-api...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📊 Metrics: prom-client (endpoint: /metrics)
   🔗 Sidecars: [prometheus, grafana]

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating metrics stack configuration...
   ✅ Created .devcontainer/prometheus/prometheus.yml
   ✅ Created .devcontainer/grafana/provisioning/datasources/prometheus.yml
   ✅ Created .devcontainer/grafana/provisioning/dashboards/provider.yml
   ✅ Created .devcontainer/grafana/provisioning/dashboards/app-metrics.json

✨ Done!
```

### Generated Configuration

When metrics libraries are detected:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    labels:
      - "prometheus.scrape=true"
      - "prometheus.port=3000"
      - "prometheus.path=/metrics"

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.retention.time=7d'

  grafana:
    image: grafana/grafana:latest
    volumes:
      - ./grafana/provisioning/datasources:/etc/grafana/provisioning/datasources:ro
      - ./grafana/provisioning/dashboards:/etc/grafana/provisioning/dashboards:ro
    ports:
      - "3001:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_AUTH_ANONYMOUS_ENABLED=true

volumes:
  prometheus-data:
  grafana-data:
```

### Accessing the Stack

| Service | URL | Credentials |
|---------|-----|-------------|
| Your App | http://localhost:3000 | - |
| Prometheus | http://localhost:9090 | - |
| Grafana | http://localhost:3001 | admin / admin |

### Pre-built Dashboard

The generated Grafana dashboard includes:

- **Request Rate**: HTTP requests per second by method and path
- **Response Time Percentiles**: p50, p95, p99 latency distribution
- **Error Rate**: 5xx error percentage over time
- **Requests by Status Code**: 2xx, 4xx, 5xx breakdown

### Adding Custom Metrics

```javascript
// Node.js with prom-client
const client = require('prom-client');

// Counter for tracking requests
const httpRequestsTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total HTTP requests',
  labelNames: ['method', 'path', 'status']
});

// Histogram for response times
const httpRequestDuration = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'path'],
  buckets: [0.01, 0.05, 0.1, 0.5, 1, 5]
});

// Expose metrics endpoint
app.get('/metrics', async (req, res) => {
  res.set('Content-Type', client.register.contentType);
  res.end(await client.register.metrics());
});
```

See [docs/sidecars/metrics.md](docs/sidecars/metrics.md) for detailed documentation.

## Generated Files

### devcontainer.json
- VS Code Dev Container configuration
- Language-specific base image or docker-compose reference
- VS Code extensions for the language
- Port forwarding
- Post-create commands

### docker-compose.yml (when services or sidecars detected)
- App service with build context
- PostgreSQL service with named volume
- Redis service with named volume
- Fluent Bit log aggregator sidecar (when logging libraries detected)
- Worker sidecar (when queue libraries detected)
- Database backup sidecar (when databases detected)
- File processor sidecar (when upload libraries detected)
- Prometheus + Grafana metrics stack (when metrics libraries detected)
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
│   │   ├── backup.go      # Database backup scripts
│   │   ├── backup_sidecar.go # Backup container generator
│   │   ├── processor_sidecar.go # File processor generator
│   │   ├── metrics_sidecar.go # Prometheus + Grafana generator
│   │   └── templates/
│   └── models/             # Data structures
└── Dockerfile              # Multi-stage container build
```

## License

MIT
