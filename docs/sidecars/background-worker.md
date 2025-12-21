# Background Worker Sidecar

The background worker sidecar pattern allows dockstart to automatically generate worker containers that run alongside your main application using the same codebase.

## Overview

When dockstart detects background job processing frameworks in your project, it can:

1. Identify the worker framework (Bull, Celery, Sidekiq, etc.)
2. Detect or infer the worker entry command
3. Generate a worker service in `docker-compose.yml`
4. Automatically add Redis/RabbitMQ dependencies

## How It Works

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│    App Container │     │  Worker Container │    │ Redis/RabbitMQ │
│    (npm start)   │────▶│  (npm run worker) │───▶│   (message bus) │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │  Same image           │  Same image
        │  Different command    │  Different command
        └───────────────────────┘
```

The key pattern is **same-image-different-command**: both the app and worker use the same Docker image, but start with different commands.

## Detected Frameworks

### Node.js

| Framework | Detection | Default Command |
|-----------|-----------|-----------------|
| Bull | `dependencies.bull` | `npm run worker` or `node worker.js` |
| BullMQ | `dependencies.bullmq` | `npm run worker` or `node worker.js` |
| Agenda | `dependencies.agenda` | `npm run worker` |

### Python

| Framework | Detection | Default Command |
|-----------|-----------|-----------------|
| Celery | `celery` in dependencies | `celery -A app worker -l INFO` |
| Dramatiq | `dramatiq` in dependencies | `dramatiq app` |
| RQ | `rq` in dependencies | `rq worker` |
| Huey | `huey` in dependencies | `huey_consumer app.huey` |

### Go

| Framework | Detection | Default Command |
|-----------|-----------|-----------------|
| Asynq | `github.com/hibiken/asynq` | `./app worker` |
| Machinery | `github.com/RichardKnop/machinery` | `./app worker` |

### Ruby

| Framework | Detection | Default Command |
|-----------|-----------|-----------------|
| Sidekiq | `sidekiq` in Gemfile | `bundle exec sidekiq` |
| Resque | `resque` in Gemfile | `bundle exec rake resque:work` |
| DelayedJob | `delayed_job` in Gemfile | `bundle exec rake jobs:work` |

## Generated Configuration

### docker-compose.yml

When a worker framework is detected, the following is added:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - ..:/workspace:cached
    command: npm start
    depends_on:
      - redis
    environment:
      - REDIS_URL=redis://redis:6379

  # Background worker sidecar
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
    healthcheck:
      test: ["CMD", "pgrep", "-f", "worker"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    restart: unless-stopped
    volumes:
      - redis-data:/data
    ports:
      - "6379:6379"

volumes:
  redis-data:
```

### Entry Point Detection

dockstart attempts to find the worker entry point in order of priority:

1. **package.json scripts** (Node.js)
   ```json
   {
     "scripts": {
       "worker": "node dist/worker.js"
     }
   }
   ```

2. **Conventional file names**
   - `worker.js`, `worker.ts` (Node.js)
   - `tasks.py`, `celery.py`, `worker.py` (Python)
   - `config/sidekiq.yml` (Ruby)

3. **Framework-specific configuration**
   - `celery.conf`, `huey_config.py` (Python)
   - `sidekiq.yml` (Ruby)

4. **Default framework command**
   - Falls back to standard framework command

## Usage

### Starting Workers

```bash
# View all containers including workers
docker compose ps

# View worker logs
docker compose logs -f worker

# Scale workers (if needed)
docker compose up -d --scale worker=3
```

### Worker Commands

```bash
# Restart just the worker
docker compose restart worker

# Stop worker only
docker compose stop worker

# Execute command in worker container
docker compose exec worker npm run queue:clear
```

## Configuration

### Worker Concurrency

Set the number of concurrent jobs:

```yaml
services:
  worker:
    environment:
      - WORKER_CONCURRENCY=4
```

Or framework-specific:

```yaml
# Celery
worker:
  command: celery -A app worker -l INFO --concurrency=4

# BullMQ
worker:
  environment:
    - BULLMQ_CONCURRENCY=10
```

### Multiple Queue Workers

For projects with multiple queues:

```yaml
services:
  worker-default:
    command: npm run worker -- --queue=default

  worker-critical:
    command: npm run worker -- --queue=critical
    deploy:
      replicas: 2
```

### Health Checks

Workers include health checks by default:

```yaml
healthcheck:
  test: ["CMD", "pgrep", "-f", "worker"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 10s
```

## Graceful Shutdown

Workers are configured for graceful shutdown:

```yaml
services:
  worker:
    stop_grace_period: 30s  # Wait for jobs to complete
    stop_signal: SIGTERM    # Send SIGTERM first
```

Ensure your worker handles signals:

```javascript
// Node.js (BullMQ)
const worker = new Worker('queue', processor);

process.on('SIGTERM', async () => {
  await worker.close();
  process.exit(0);
});
```

```python
# Python (Celery)
# Celery handles SIGTERM automatically with --pool=prefork
celery -A app worker --pool=prefork
```

## Troubleshooting

### Worker Not Starting

1. Check if Redis is running:
   ```bash
   docker compose ps redis
   ```

2. View worker logs:
   ```bash
   docker compose logs worker
   ```

3. Verify the worker command:
   ```bash
   docker compose config | grep -A5 worker
   ```

### Jobs Not Processing

1. Check Redis connection:
   ```bash
   docker compose exec worker redis-cli ping
   ```

2. Verify queue has jobs:
   ```bash
   # For Bull/BullMQ
   docker compose exec worker npx bull-repl

   # For Celery
   docker compose exec worker celery -A app inspect active
   ```

### Worker Crashes on Job

1. Enable debug logging:
   ```yaml
   environment:
     - LOG_LEVEL=debug
   ```

2. Check for OOM issues:
   ```yaml
   deploy:
     resources:
       limits:
         memory: 1G
   ```

## Architecture Decision

See [ADR-002: Background Worker Sidecar Architecture](../adr/002-worker-sidecar-architecture.md) for the rationale behind this design.

## Related

- [Redis Queue Documentation](https://redis.io/docs/data-types/lists/)
- [BullMQ Documentation](https://docs.bullmq.io/)
- [Celery Documentation](https://docs.celeryq.dev/)
- [Docker Compose Scaling](https://docs.docker.com/compose/compose-file/deploy/)
