# Log Aggregator Sidecar

The log aggregator sidecar provides centralized logging for your development environment using **Fluent Bit**, a lightweight and high-performance log processor.

## Overview

When dockstart detects structured logging libraries in your project, it automatically:

1. Generates a `fluent-bit.conf` configuration file
2. Adds a Fluent Bit service to `docker-compose.yml`
3. Configures your app container to use the Docker fluentd logging driver
4. Forwards port 24224 in `devcontainer.json`

## How It Works

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  App Container  │────▶│   Fluent Bit    │────▶│     stdout      │
│  (your code)    │     │   (sidecar)     │     │   (viewable)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │
        │  Docker fluentd       │  Enriched logs
        │  logging driver       │  with metadata
        │                       │
```

## Detected Libraries

dockstart automatically detects these structured logging libraries:

### Node.js

| Library | Default Format | Detection |
|---------|---------------|-----------|
| pino | JSON | `dependencies.pino` |
| bunyan | JSON | `dependencies.bunyan` |
| winston | configurable | `dependencies.winston` |
| morgan | text | `dependencies.morgan` |
| log4js | configurable | `dependencies.log4js` |

### Go

| Library | Default Format | Detection |
|---------|---------------|-----------|
| zap | JSON | `go.uber.org/zap` |
| zerolog | JSON | `github.com/rs/zerolog` |
| logrus | text | `github.com/sirupsen/logrus` |
| slog | JSON | `log/slog` |

### Python

| Library | Default Format | Detection |
|---------|---------------|-----------|
| structlog | JSON | `structlog` in dependencies |
| python-json-logger | JSON | `python-json-logger` in dependencies |
| loguru | text | `loguru` in dependencies |
| logbook | text | `logbook` in dependencies |

### Rust

| Library | Default Format | Detection |
|---------|---------------|-----------|
| tracing | JSON | `tracing` in Cargo.toml |
| slog | JSON | `slog` in Cargo.toml |
| log | text | `log` in Cargo.toml |
| env_logger | text | `env_logger` in Cargo.toml |

## Generated Files

### fluent-bit.conf

Located at `.devcontainer/fluent-bit.conf`:

```ini
[SERVICE]
    Flush           1
    Log_Level       info
    Daemon          off
    Parsers_File    /fluent-bit/etc/parsers.conf

[INPUT]
    Name            forward
    Listen          0.0.0.0
    Port            24224
    Tag             docker.*

# Only included when LogFormat is "json"
[FILTER]
    Name            parser
    Match           docker.*
    Key_Name        log
    Parser          json
    Reserve_Data    On

[FILTER]
    Name            modify
    Match           *
    Add             environment development
    Add             project my-app

[OUTPUT]
    Name            stdout
    Match           *
    Format          json_lines
```

### docker-compose.yml additions

```yaml
services:
  app:
    # ... existing config ...
    depends_on:
      - fluent-bit
    environment:
      - LOG_LEVEL=debug
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: app.my-app
        fluentd-async: "true"

  fluent-bit:
    image: fluent/fluent-bit:latest
    restart: unless-stopped
    volumes:
      - ./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf:ro
    ports:
      - "24224:24224"
      - "24224:24224/udp"

volumes:
  fluent-bit-logs:
```

## Usage

### Viewing Logs

```bash
# View all logs (app + fluent-bit)
docker compose logs -f

# View only fluent-bit processed logs
docker compose logs -f fluent-bit

# View only app logs (raw, before processing)
docker compose logs -f app
```

### Log Format

Logs are output in JSON Lines format:

```json
{"date":1703001234.567,"environment":"development","project":"my-app","level":"info","msg":"Server started","port":3000}
{"date":1703001235.123,"environment":"development","project":"my-app","level":"debug","msg":"Request received","path":"/api/users"}
```

### Filtering Logs

Use `jq` to filter logs:

```bash
# Filter by log level
docker compose logs -f fluent-bit | jq 'select(.level == "error")'

# Filter by message content
docker compose logs -f fluent-bit | jq 'select(.msg | contains("database"))'

# Pretty print
docker compose logs -f fluent-bit | jq .
```

## Customization

### Disabling Log Sidecar

Currently, the log sidecar is automatically enabled when logging libraries are detected. To disable it, you can manually remove the Fluent Bit service from the generated `docker-compose.yml`.

### Custom Fluent Bit Configuration

After generation, you can modify `.devcontainer/fluent-bit.conf` to:

- Add additional outputs (Elasticsearch, Loki, etc.)
- Add custom parsers
- Modify log enrichment
- Change output format

Example: Adding Elasticsearch output:

```ini
[OUTPUT]
    Name            es
    Match           *
    Host            elasticsearch
    Port            9200
    Index           app-logs
    Type            _doc
```

## Troubleshooting

### Logs Not Appearing

1. Check Fluent Bit is running:
   ```bash
   docker compose ps fluent-bit
   ```

2. Check Fluent Bit logs for errors:
   ```bash
   docker compose logs fluent-bit
   ```

3. Verify the fluentd driver is configured:
   ```bash
   docker inspect <app-container> | jq '.[0].HostConfig.LogConfig'
   ```

### Container Won't Start

If the app container won't start with "Cannot connect to fluentd", ensure:

1. Fluent Bit service is listed in `depends_on`
2. `fluentd-async: "true"` is set in logging options

### JSON Parsing Errors

If logs aren't being parsed as JSON:

1. Verify your app is outputting valid JSON
2. Check the `LogFormat` detection matches your app's output
3. Add custom parser rules if needed

## Architecture Decision

See [ADR-001: Log Aggregator Sidecar Architecture](../adr/001-log-aggregator-sidecar.md) for the rationale behind choosing Fluent Bit over alternatives like Vector or Filebeat.

## Related

- [Fluent Bit Documentation](https://docs.fluentbit.io/)
- [Docker Logging Drivers](https://docs.docker.com/engine/logging/drivers/)
- [Docker Compose Logging Configuration](https://docs.docker.com/compose/compose-file/compose-file-v3/#logging)
