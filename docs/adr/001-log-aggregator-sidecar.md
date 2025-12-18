# ADR-001: Log Aggregator Sidecar Architecture

**Status**: Accepted
**Date**: 2025-12-18
**Issue**: #22

## Context

Dockstart generates Docker dev environments. To provide better observability in development, we want to add optional log aggregation sidecars that collect and centralize logs from application containers.

### Problem Statement

Developers need to:
1. View logs from multiple containers in one place
2. Search and filter logs during development
3. Optionally forward logs to external services (Loki, Elasticsearch)

## Research: Log Aggregator Comparison

### Candidates Evaluated

| Feature | Fluent Bit | Vector | Filebeat |
|---------|------------|--------|----------|
| **Language** | C | Rust | Go |
| **Memory Usage** | ~26MB | ~30MB | ~80MB |
| **CPU Usage** | Low (~27%) | Very Low | Moderate |
| **Config Format** | INI/YAML | YAML/TOML | YAML |
| **Docker Integration** | Native driver | Good | Good (ELK focus) |
| **Learning Curve** | Moderate | Steeper (VRL) | Easy |
| **Plugin Ecosystem** | Large | Growing | ELK-centric |

### Performance Benchmarks

Based on tests forwarding 5,000 1KB events/second:
- **Fluent Bit**: 27% CPU, 26MB memory
- **Fluentd**: 80% CPU, 120MB memory
- **Vector**: Competitive with Fluent Bit, lower memory in some tests

### Recommendation: **Fluent Bit**

**Rationale**:
1. **Lightweight**: Critical for dev machines running multiple containers
2. **Native Docker logging driver**: Simplest integration path
3. **CNCF Project**: Well-maintained, vendor-neutral
4. **Good defaults**: Works out-of-box for common scenarios
5. **Kubernetes-ready**: Skills transfer to production environments

## Decision

### Primary Choice: Fluent Bit

Use Fluent Bit as the default log aggregator sidecar for dockstart-generated environments.

### Alternative: Vector

Offer Vector as an alternative for users who need:
- Complex log transformations (VRL)
- Higher throughput requirements
- Rust ecosystem preference

## Architecture

### Log Collection Strategies

#### Strategy 1: Docker Log Driver (Recommended for Dev)

```yaml
services:
  app:
    logging:
      driver: fluentd
      options:
        fluentd-address: localhost:24224
        tag: app.{{.Name}}

  fluent-bit:
    image: fluent/fluent-bit:latest
    ports:
      - "24224:24224"
    volumes:
      - ./fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf
```

**Pros**: Native integration, no app changes needed
**Cons**: Container won't start if Fluent Bit unavailable

#### Strategy 2: Shared Volume (Fallback)

```yaml
services:
  app:
    volumes:
      - app-logs:/var/log/app

  fluent-bit:
    volumes:
      - app-logs:/var/log/app:ro

volumes:
  app-logs:
```

**Pros**: Decoupled, app starts independently
**Cons**: Requires apps to write to files, not stdout

#### Strategy 3: Docker Socket (Advanced)

```yaml
services:
  fluent-bit:
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

**Pros**: Collects from all containers automatically
**Cons**: Security implications, more complex

### Chosen Strategy: Docker Log Driver

For dev environments, Strategy 1 (Docker Log Driver) provides:
- Zero app code changes
- Automatic stdout/stderr capture
- Simple configuration
- Good developer experience

## Detection Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Analyze Project │────▶│ Detect Logging   │────▶│ Generate        │
│ Dependencies    │     │ Libraries        │     │ Sidecar Config  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

### Logging Library Detection

| Language | Libraries to Detect | Indicates |
|----------|---------------------|-----------|
| Node.js | winston, pino, bunyan, morgan | Structured logging |
| Go | zap, zerolog, logrus, slog | Structured logging |
| Python | structlog, loguru, python-json-logger | Structured logging |
| Rust | tracing, log, slog | Structured logging |

### Detection Logic

```go
// Pseudo-code for log sidecar detection
func ShouldGenerateLogSidecar(detection *Detection) bool {
    // Option 1: Always generate (user can opt-out)
    // Option 2: Only if structured logging library detected
    // Option 3: User explicitly requests via flag

    return hasStructuredLoggingLibrary(detection) ||
           userRequestedLogs(detection.Options)
}
```

## Configuration Schema

### User-Facing Options

```yaml
# .dockstart.yml (future config file)
sidecars:
  logs:
    enabled: true           # Enable log aggregation
    provider: fluent-bit    # fluent-bit | vector
    outputs:
      - stdout              # Always available
      - file                # /var/log/dockstart/
      - loki                # Optional: Grafana Loki
    retention: 7d           # Log retention period
```

### Generated Fluent Bit Config

```ini
[SERVICE]
    Flush        1
    Log_Level    info
    Parsers_File parsers.conf

[INPUT]
    Name         forward
    Listen       0.0.0.0
    Port         24224

[OUTPUT]
    Name         stdout
    Match        *
    Format       json_lines

[OUTPUT]
    Name         file
    Match        *
    Path         /var/log/dockstart/
    File         app.log
```

## Docker Concepts Reference

### Log Drivers

| Driver | Use Case | docker logs Support |
|--------|----------|---------------------|
| json-file | Default, local storage | Yes |
| local | Better performance than json-file | Yes |
| fluentd | Forward to Fluentd/Fluent Bit | No |
| syslog | System syslog integration | No |
| none | Disable logging | No |

### Volume Types for Logging

| Type | Use Case | Example |
|------|----------|---------|
| Named Volume | Persistent logs | `logs:/var/log` |
| Bind Mount | Dev access to logs | `./logs:/var/log` |
| tmpfs | Ephemeral logs | `type: tmpfs` |

### Best Practices for Dev Logging

1. **Log Rotation**: Set `max-size` and `max-file` to prevent disk fill
2. **Structured Logging**: Use JSON format for easier parsing
3. **Consistent Timestamps**: UTC, ISO8601 format
4. **Log Levels**: DEBUG in dev, INFO+ in staging/prod
5. **No Sensitive Data**: Never log passwords, tokens, PII

## Implementation Plan

### Phase 1: Basic Fluent Bit Sidecar
- [ ] Detect structured logging libraries (#23)
- [ ] Create Fluent Bit sidecar template (#24)
- [ ] Update compose generator (#25)

### Phase 2: Enhanced Features
- [ ] Add Vector as alternative provider
- [ ] Support Loki/Elasticsearch outputs
- [ ] Add log viewer UI recommendation

## Consequences

### Positive
- Developers get centralized logging without configuration
- Logs persist across container restarts
- Foundation for production logging patterns

### Negative
- Additional container resource usage (~30MB RAM)
- Slight increase in generated docker-compose complexity
- Learning curve for Fluent Bit configuration

### Neutral
- Opinionated choice (Fluent Bit) may not suit all users
- Some users may prefer their existing logging setup

## References

- [Fluent Bit Documentation](https://docs.fluentbit.io/)
- [Vector Documentation](https://vector.dev/docs/)
- [Docker Logging Drivers](https://docs.docker.com/engine/logging/drivers/)
- [CNCF Log Collector Comparison](https://www.cncf.io/blog/2022/02/10/logstash-fluentd-fluent-bit-or-vector-how-to-choose-the-right-open-source-log-collector/)
- [Docker Logging Best Practices - Datadog](https://www.datadoghq.com/blog/docker-logging/)
- [Better Stack Log Shippers Guide](https://betterstack.com/community/guides/logging/log-shippers-explained/)
