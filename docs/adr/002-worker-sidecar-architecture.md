# ADR-002: Background Worker Sidecar Architecture

**Status**: Accepted
**Date**: 2025-12-19
**Issue**: #29

## Context

Dockstart generates Docker dev environments. Many applications require background workers for async job processing. We want to auto-detect worker frameworks and generate appropriate worker sidecar containers.

### Problem Statement

Developers need to:
1. Run background workers alongside their application
2. Use the same codebase/image but different entry commands
3. Scale workers independently from the main app
4. See worker logs alongside application logs

## Research: Worker Framework Patterns

### Candidates Evaluated

| Framework | Language | Redis Req | Detection | Worker Command |
|-----------|----------|-----------|-----------|----------------|
| **Bull/BullMQ** | Node.js | Yes | `dependencies.bull` or `dependencies.bullmq` | `node worker.js` or `npm run worker` |
| **Celery** | Python | Yes (or RabbitMQ) | `celery` in dependencies | `celery -A app worker` |
| **Sidekiq** | Ruby | Yes | `sidekiq` in Gemfile | `bundle exec sidekiq` |
| **Asynq** | Go | Yes | `github.com/hibiken/asynq` | Same binary, different flag |
| **Tokio** | Rust | Optional | `tokio` with queue crate | Same binary, different command |
| **Dramatiq** | Python | Yes (or RabbitMQ) | `dramatiq` in dependencies | `dramatiq app` |
| **RQ (Redis Queue)** | Python | Yes | `rq` in dependencies | `rq worker` |
| **Faktory** | Multi-lang | Yes (Faktory server) | `faktory_worker_*` | Language-specific |

### Worker Startup Patterns

#### Pattern 1: Separate Entry File (Node.js/Bull)

```javascript
// worker.js
const { Worker } = require('bullmq');
new Worker('queue', async job => { ... });
```

**Command**: `node worker.js` or `npm run worker`

#### Pattern 2: Framework CLI (Python/Celery)

```python
# tasks.py
from celery import Celery
app = Celery('tasks', broker='redis://redis:6379')

@app.task
def process(data): ...
```

**Command**: `celery -A tasks worker --loglevel=INFO`

#### Pattern 3: Same Binary, Different Flag (Go/Asynq)

```go
func main() {
    if os.Args[1] == "worker" {
        srv := asynq.NewServer(...)
        srv.Run(mux)
    } else {
        // HTTP server
    }
}
```

**Command**: `./app worker` or environment-based detection

#### Pattern 4: Gem/Bundle Runner (Ruby/Sidekiq)

```ruby
# Gemfile
gem 'sidekiq'

# config/sidekiq.yml
:queues:
  - default
  - critical
```

**Command**: `bundle exec sidekiq -C config/sidekiq.yml`

### Worker Entry Point Detection

| Framework | Entry Point Detection |
|-----------|----------------------|
| Bull/BullMQ | Check `scripts.worker` in package.json, or `worker.js`/`worker.ts` file |
| Celery | Check for `celery.py`, `tasks.py`, or celery app in `__init__.py` |
| Sidekiq | Check `config/sidekiq.yml` or `sidekiq.rb` |
| Asynq | Parse main.go for asynq import and worker setup |
| RQ | Check for `rq` script in pyproject.toml or rq config |

## Decision

### Worker Detection Strategy

Use a two-tier detection approach:

1. **Framework Detection**: Identify worker frameworks from dependencies
2. **Entry Point Discovery**: Find or infer the worker entry command

### Same-Image-Different-Command Pattern

Workers reuse the application image but override the command:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    command: npm start  # Main app

  worker:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    command: npm run worker  # Background worker
    depends_on:
      - app
      - redis
```

**Rationale**:
- Single build, multiple services
- Consistent dependencies
- Simpler CI/CD
- Better cache utilization

### Worker Command Configuration

#### Default Commands by Framework

```go
var WorkerCommands = map[string]map[string]WorkerConfig{
    "node": {
        "bull":    {Command: "npm run worker", FallbackCmd: "node worker.js"},
        "bullmq":  {Command: "npm run worker", FallbackCmd: "node worker.js"},
    },
    "python": {
        "celery":   {Command: "celery -A ${APP_NAME} worker -l INFO", DetectApp: true},
        "dramatiq": {Command: "dramatiq ${APP_NAME}", DetectApp: true},
        "rq":       {Command: "rq worker", FallbackCmd: "rq worker -u redis://redis:6379"},
    },
    "go": {
        "asynq":    {Command: "./${BINARY} worker", DetectBinary: true},
    },
    "ruby": {
        "sidekiq":  {Command: "bundle exec sidekiq", ConfigFile: "config/sidekiq.yml"},
    },
}
```

#### Entry Point Discovery

1. Check `package.json` scripts for `worker` or `queue` scripts
2. Look for conventional files: `worker.js`, `worker.ts`, `tasks.py`, `celery.py`
3. Parse framework-specific configs: `sidekiq.yml`, `celery.conf`
4. Fall back to framework default command

## Architecture

### Detection Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Analyze Project │────▶│ Detect Worker    │────▶│ Find Entry      │
│ Dependencies    │     │ Frameworks       │     │ Point/Command   │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                          │
                                                          ▼
                              ┌─────────────────────────────────────┐
                              │     Generate Worker Sidecar         │
                              │  - Same Dockerfile                  │
                              │  - Override command                 │
                              │  - Add redis dependency             │
                              │  - Configure environment            │
                              └─────────────────────────────────────┘
```

### Generated Docker Compose Structure

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

  # Worker sidecar - uses same image, different command
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
    deploy:
      replicas: 1
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

### Worker Configuration Options

```go
type WorkerSidecarConfig struct {
    // Framework is the detected worker framework (bull, celery, etc.)
    Framework string

    // Command is the worker startup command
    Command string

    // Concurrency is the number of worker threads/processes
    Concurrency int

    // Queues is the list of queues to process (optional)
    Queues []string

    // RequiresRedis indicates if Redis dependency should be added
    RequiresRedis bool

    // RequiresRabbitMQ indicates if RabbitMQ dependency should be added
    RequiresRabbitMQ bool

    // HealthCheck is the healthcheck configuration
    HealthCheck WorkerHealthCheck
}
```

## Docker Concepts Reference

### Command Override

The `command:` directive overrides the Dockerfile's `CMD`:

```yaml
# Dockerfile: CMD ["npm", "start"]
# docker-compose.yml:
services:
  worker:
    command: npm run worker  # Overrides CMD
```

### Service Replicas (Docker Compose v3+)

```yaml
services:
  worker:
    deploy:
      replicas: 3  # Run 3 worker instances
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
```

### Worker Healthchecks

```yaml
services:
  worker:
    healthcheck:
      test: ["CMD", "pgrep", "-f", "worker"]  # Check worker process
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
```

### Graceful Shutdown

```yaml
services:
  worker:
    stop_grace_period: 30s  # Wait for job completion
    stop_signal: SIGTERM    # Send SIGTERM first
```

Workers should handle signals:

```javascript
// Node.js graceful shutdown
process.on('SIGTERM', async () => {
  await worker.close();
  process.exit(0);
});
```

```python
# Celery graceful shutdown
celery -A app worker --pool=prefork --concurrency=4
# Celery handles SIGTERM automatically
```

## Implementation Plan

### Phase 1: Core Detection
- [ ] Add worker framework fields to Detection model (#30)
- [ ] Implement worker framework detection per language (#31)

### Phase 2: Worker Sidecar Generation
- [ ] Create worker sidecar compose template (#32)
- [ ] Add worker command inference logic
- [ ] Update compose generator

### Phase 3: Advanced Features
- [ ] Support multiple worker types per project
- [ ] Add queue-specific workers
- [ ] Implement worker scaling configuration

## Consequences

### Positive
- Developers get workers without manual docker-compose setup
- Same-image pattern ensures consistency
- Foundation for production worker deployments
- Workers automatically get Redis/RabbitMQ dependencies

### Negative
- Additional container resource usage
- More complex generated docker-compose
- Need to handle various worker entry patterns
- May require convention over configuration

### Neutral
- Workers use same Dockerfile as app (by design)
- Worker scaling is limited in dev (by design)
- Some frameworks may need manual command override

## References

- [Bull/BullMQ Documentation](https://docs.bullmq.io/)
- [Celery Documentation](https://docs.celeryq.dev/)
- [Sidekiq Wiki](https://github.com/mperham/sidekiq/wiki)
- [Asynq Documentation](https://github.com/hibiken/asynq)
- [Docker Compose Deploy](https://docs.docker.com/compose/compose-file/deploy/)
- [Docker Healthcheck](https://docs.docker.com/engine/reference/builder/#healthcheck)
- [Graceful Shutdown in Docker](https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-terminating-with-grace)
