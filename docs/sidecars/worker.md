# Worker Sidecar

Quick reference for the background worker sidecar feature. For full documentation, see [background-worker.md](background-worker.md).

## Quick Start

When dockstart detects a queue library in your project, it automatically generates a worker sidecar.

### Detected Libraries

| Language | Libraries |
|----------|-----------|
| Node.js | bull, bullmq, bee-queue, agenda, kue, pg-boss |
| Python | celery, rq, dramatiq, huey, arq, taskiq |
| Go | asynq, machinery, gocraft-work, rmq, gocelery |
| Rust | sidekiq, celery, lapin, apalis, faktory |

### Example Output

```bash
$ dockstart ./my-api

📂 Analyzing ./my-api...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [redis]
   👷 Worker: bull (command: npm run worker)
```

## Generated Configuration

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    # ... app config ...

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

## Common Commands

```bash
# View worker logs
docker compose logs -f worker

# Scale to 3 workers
docker compose up -d --scale worker=3

# Restart workers
docker compose restart worker

# Execute command in worker
docker compose exec worker npm run queue:status
```

## Auto-Redis

Redis is automatically added when using Redis-based queue libraries:
- Node.js: bull, bullmq, bee-queue
- Go: asynq, rmq
- Python: rq, arq
- Rust: sidekiq

## See Also

- [Full Worker Documentation](background-worker.md)
- [ADR-002: Worker Architecture](../adr/002-worker-sidecar-architecture.md)
- [Example Project](../examples/worker/)
