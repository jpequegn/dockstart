# ADR-006: Distributed Tracing Sidecar Architecture

**Status**: Proposed
**Date**: 2025-12-21
**Issue**: #64

## Context

Dockstart generates Docker dev environments. To provide complete observability, we want to add distributed tracing capabilities that help developers understand request flows across services and identify performance bottlenecks.

### Problem Statement

Developers need to:
1. Trace requests across multiple microservices
2. Identify latency bottlenecks in request processing
3. Debug timing issues and service dependencies
4. Learn production tracing patterns in a local environment
5. Correlate traces with logs and metrics (observability triad)

## Research: Protocol Comparison

### OpenTelemetry (OTLP) vs Jaeger Native

| Feature | OTLP | Jaeger Native |
|---------|------|---------------|
| **Standard** | CNCF standard, vendor-neutral | Jaeger-specific |
| **Adoption** | Industry-wide, growing rapidly | Jaeger ecosystem only |
| **SDK Support** | All major languages | Limited to Jaeger SDKs |
| **Future-Proof** | Yes, unified telemetry | Jaeger moving to OTLP |
| **Protocol** | gRPC (4317) / HTTP (4318) | Thrift (14268) / UDP (6831) |
| **Signals** | Traces, Metrics, Logs | Traces only |

### Recommendation: **OTLP (OpenTelemetry Protocol)**

**Rationale**:
1. **Industry Standard**: CNCF graduated project, vendor-neutral
2. **Unified Telemetry**: Same protocol for traces, metrics, and logs
3. **Jaeger Support**: Jaeger natively supports OTLP since v1.35
4. **SDK Availability**: OpenTelemetry SDKs for all major languages
5. **Future-Proof**: Jaeger is deprecating native protocols in favor of OTLP

## Research: Tracing Backend Comparison

### Candidates Evaluated

| Feature | Jaeger | Zipkin | Tempo |
|---------|--------|--------|-------|
| **License** | Apache 2.0 | Apache 2.0 | AGPL |
| **Memory Usage** | ~100MB (all-in-one) | ~200MB | ~50MB |
| **Storage** | Memory/Badger/Cassandra/ES | Memory/MySQL/ES | Object storage |
| **UI Quality** | Excellent (trace analysis) | Good | Via Grafana |
| **Docker Integration** | Excellent | Excellent | Excellent |
| **Learning Value** | High (K8s standard) | Moderate | Moderate |
| **OTLP Support** | Native | Via Collector | Native |
| **Comparison View** | Yes | Yes | Via Grafana |

### Recommendation: **Jaeger All-in-One**

**Rationale**:
1. **Industry Standard**: Widely adopted in Kubernetes ecosystems
2. **Low Resource**: All-in-one mode uses ~100MB RAM
3. **Excellent UI**: Built-in trace analysis, service graphs
4. **OTLP Native**: Full support for OpenTelemetry protocol
5. **Simple Setup**: Single container, no external dependencies
6. **CNCF Graduated**: Mature, production-proven project

## Jaeger Deployment Modes

### All-in-One (Development)

Single container with all components:
- Collector (receives spans)
- Query (API + UI)
- In-memory storage

```yaml
services:
  jaeger:
    image: jaegertracing/jaeger:2.1.0
    container_name: jaeger
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"  # UI
      - "4317:4317"    # OTLP gRPC
      - "4318:4318"    # OTLP HTTP
    restart: unless-stopped
```

**Pros**: Simple, low overhead, perfect for development
**Cons**: In-memory storage (data lost on restart)

### Distributed (Production)

Separate components with persistent storage:
- Collector → Kafka → Ingester → Elasticsearch/Cassandra
- Query service for UI

**Not recommended for dockstart**: Adds significant complexity without benefit for local development.

### Decision: **All-in-One Mode**

For development environments, all-in-one is the clear choice:
- Single container (~100MB RAM)
- Zero external dependencies
- Data persistence not critical for dev
- Can upgrade to distributed if needed

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Docker Compose Network                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌─────────────┐    OTLP gRPC :4317     ┌──────────────────────┐   │
│   │             │──────────────────────►│                      │   │
│   │     App     │                        │       Jaeger         │   │
│   │             │◄──────────────────────│   (all-in-one)       │   │
│   └─────────────┘    trace context       │      :16686          │   │
│                                          └──────────────────────┘   │
│                                                    ▲                │
│   ┌─────────────┐    OTLP HTTP :4318              │                │
│   │             │─────────────────────────────────┤                │
│   │   Worker    │                                 │                │
│   │             │                                 │                │
│   └─────────────┘                                 │                │
│                                                   │                │
│   ┌─────────────┐    OTLP gRPC :4317             │                │
│   │             │─────────────────────────────────┘                │
│   │   Service   │                                                  │
│   │      B      │                                                  │
│   └─────────────┘                                                  │
│                                                                      │
│   Trace context propagated via HTTP headers (traceparent, b3)      │
└─────────────────────────────────────────────────────────────────────┘
```

### Trace Collection Protocols

| Protocol | Port | Transport | Use Case |
|----------|------|-----------|----------|
| OTLP gRPC | 4317 | Protobuf/gRPC | OpenTelemetry SDKs (recommended) |
| OTLP HTTP | 4318 | JSON/Protobuf | HTTP-only environments, browsers |
| Jaeger Thrift | 14268 | HTTP/Thrift | Legacy Jaeger clients |
| Jaeger Compact | 6831 | UDP/Thrift | Ultra-low overhead |
| Zipkin | 9411 | JSON | Zipkin compatibility |

### Recommended: OTLP gRPC (4317)

- Highest performance (binary protocol)
- Full OpenTelemetry feature support
- Automatic retry and batching
- Fallback to HTTP (4318) for restricted environments

## Environment Variable Injection

### OpenTelemetry SDK Configuration

```yaml
services:
  app:
    environment:
      # Required
      - OTEL_SERVICE_NAME=${PROJECT_NAME}
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317

      # Optional but recommended
      - OTEL_TRACES_EXPORTER=otlp
      - OTEL_METRICS_EXPORTER=none          # Metrics via Prometheus
      - OTEL_LOGS_EXPORTER=none             # Logs via Fluent Bit
      - OTEL_TRACES_SAMPLER=parentbased_always_on
      - OTEL_RESOURCE_ATTRIBUTES=deployment.environment=development
```

### Language-Specific Configuration

| Language | SDK Package | Auto-Instrumentation |
|----------|-------------|---------------------|
| Node.js | @opentelemetry/sdk-node | @opentelemetry/auto-instrumentations-node |
| Go | go.opentelemetry.io/otel | otelgin, otelhttp, etc. |
| Python | opentelemetry-sdk | opentelemetry-instrumentation |
| Rust | opentelemetry | tracing-opentelemetry |
| Java | opentelemetry-java-agent | Automatic (agent) |

### Trace Context Propagation

For distributed tracing across services, context must be propagated:

```yaml
environment:
  - OTEL_PROPAGATORS=tracecontext,baggage,b3multi
```

| Propagator | Header | Use When |
|------------|--------|----------|
| tracecontext | traceparent, tracestate | W3C standard (recommended) |
| b3multi | X-B3-TraceId, X-B3-SpanId | Zipkin compatibility |
| baggage | baggage | Cross-service metadata |

## Sampling Strategies

### Development Environment Recommendation

| Strategy | Rate | Use Case |
|----------|------|----------|
| always_on | 100% | Default for development |
| parentbased_always_on | 100% (respects parent) | Multi-service development |
| traceidratio | 10-50% | High-volume development |
| always_off | 0% | Disable tracing temporarily |

### Decision: `parentbased_always_on` (100%)

**Rationale**:
1. **Full Visibility**: See every request in development
2. **Low Volume**: Dev environments have low traffic
3. **Parent Respect**: Honors upstream sampling decisions
4. **Debugging**: Complete traces for troubleshooting

```yaml
environment:
  - OTEL_TRACES_SAMPLER=parentbased_always_on
```

### Production Note

For production environments (not dockstart's scope), use probabilistic sampling:
```yaml
environment:
  - OTEL_TRACES_SAMPLER=parentbased_traceidratio
  - OTEL_TRACES_SAMPLER_ARG=0.1  # 10% sampling
```

## Tracing Library Detection

### Libraries to Detect

| Language | Libraries | Auto-Instrumentation Available |
|----------|-----------|-------------------------------|
| Node.js | @opentelemetry/sdk-node, @opentelemetry/api | Yes |
| Go | go.opentelemetry.io/otel | Partial (per-framework) |
| Python | opentelemetry-sdk, opentelemetry-api | Yes |
| Rust | opentelemetry, tracing-opentelemetry | No |
| Java | opentelemetry-api | Yes (agent) |

### Detection Model Extension

```go
// Addition to Detection struct in models/project.go
type Detection struct {
    // ... existing fields ...

    // TracingLibraries is a list of detected OpenTelemetry/tracing libraries
    TracingLibraries []string

    // TracingProtocol is the detected or inferred protocol
    // Values: "otlp", "jaeger", "zipkin", "unknown"
    TracingProtocol string
}

// NeedsTracing returns true if tracing libraries were detected
func (d *Detection) NeedsTracing() bool {
    return len(d.TracingLibraries) > 0
}
```

## Jaeger UI Features

The Jaeger UI provides powerful debugging capabilities:

### Trace Analysis
- Request timeline visualization
- Span duration breakdown
- Error highlighting
- Log correlation

### Service Graph
- Automatic service dependency mapping
- Traffic flow visualization
- Error rate indicators

### Trace Comparison
- Compare two traces side-by-side
- Identify performance regressions
- A/B testing support

## Integration with Existing Sidecars

### Observability Triad

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Metrics   │     │    Logs     │     │   Traces    │
│ (Prometheus)│     │ (Fluent Bit)│     │  (Jaeger)   │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       └───────────────────┴───────────────────┘
                           │
                    ┌──────┴──────┐
                    │   Grafana   │
                    │  Dashboard  │
                    └─────────────┘
```

### Grafana Integration

Grafana can query Jaeger for trace visualization:

```yaml
datasources:
  - name: Jaeger
    type: jaeger
    access: proxy
    url: http://jaeger:16686
    editable: false
```

### Trace-to-Log Correlation

Link traces with logs using trace IDs:

```yaml
# Fluent Bit parser for trace context
[PARSER]
    Name   json_with_trace
    Format json
    Time_Key time
    Decode_Field_As json trace_id
```

## Configuration Schema

### User-Facing Options (Future)

```yaml
# .dockstart.yml (future config file)
sidecars:
  tracing:
    enabled: true              # Enable tracing sidecar
    backend: jaeger            # jaeger | zipkin | tempo
    protocol: otlp             # otlp | jaeger | zipkin
    sampling: always_on        # always_on | traceidratio
    sampling_rate: 1.0         # For traceidratio sampler
    ui_port: 16686             # Jaeger UI port
```

### Generated Docker Compose

```yaml
services:
  jaeger:
    image: jaegertracing/jaeger:2.1.0
    container_name: jaeger
    environment:
      - COLLECTOR_OTLP_ENABLED=true
    ports:
      - "16686:16686"    # UI
      - "4317:4317"      # OTLP gRPC
      - "4318:4318"      # OTLP HTTP
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:16686/"]
      interval: 30s
      timeout: 10s
      retries: 3
    restart: unless-stopped
    networks:
      - default

  app:
    environment:
      - OTEL_SERVICE_NAME=${PROJECT_NAME:-app}
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4317
      - OTEL_TRACES_SAMPLER=parentbased_always_on
      - OTEL_PROPAGATORS=tracecontext,baggage
    depends_on:
      jaeger:
        condition: service_healthy
```

## Docker Concepts Reference

### Multi-Port Container Configuration

```yaml
jaeger:
  ports:
    - "16686:16686"  # UI (HTTP)
    - "4317:4317"    # OTLP gRPC
    - "4318:4318"    # OTLP HTTP
    - "6831:6831/udp"  # Jaeger compact (UDP)
```

### Protocol Differences

| Protocol | Transport | Overhead | Use Case |
|----------|-----------|----------|----------|
| gRPC | HTTP/2, binary | Low | Service-to-service |
| HTTP | HTTP/1.1, JSON | Medium | Browsers, proxies |
| UDP | Datagram | Minimal | Fire-and-forget |

### Service Mesh Patterns

For trace propagation in service meshes:
1. Extract trace context from incoming request headers
2. Store in thread-local/context storage
3. Inject into outgoing request headers
4. SDKs handle this automatically with proper configuration

## Implementation Plan

### Phase 1: Tracing Library Detection (Issue #65)
- [ ] Detect OpenTelemetry SDK in Node.js dependencies
- [ ] Detect go.opentelemetry.io/otel in Go modules
- [ ] Detect opentelemetry-sdk in Python requirements
- [ ] Detect opentelemetry crate in Rust Cargo.toml
- [ ] Add TracingLibraries field to Detection model

### Phase 2: Jaeger Sidecar Generation (Issue #66)
- [ ] Create Jaeger service template for docker-compose
- [ ] Generate OTEL environment variable injection
- [ ] Add health check configuration
- [ ] Integrate with devcontainer port forwarding

### Phase 3: Documentation & Testing (Issue #67)
- [ ] Create sidecar documentation (docs/sidecars/tracing.md)
- [ ] Add integration tests
- [ ] Create example project with tracing
- [ ] Write getting started guide

## Consequences

### Positive
- Complete observability stack (metrics + logs + traces)
- Developers learn production-grade tracing patterns
- Easy debugging of distributed request flows
- Low overhead (~100MB RAM for Jaeger)
- OTLP standardization ensures vendor flexibility

### Negative
- Additional container in docker-compose
- Requires OpenTelemetry SDK in application code
- Learning curve for distributed tracing concepts
- Auto-instrumentation varies by language

### Neutral
- Jaeger is opinionated choice (alternatives: Zipkin, Tempo)
- 100% sampling may not reflect production behavior
- In-memory storage loses data on restart (acceptable for dev)

## Zipkin Compatibility

For projects already using Zipkin:

```yaml
environment:
  # Zipkin-compatible exporters
  - OTEL_EXPORTER_ZIPKIN_ENDPOINT=http://jaeger:9411/api/v2/spans
  - OTEL_TRACES_EXPORTER=zipkin
```

Jaeger's Zipkin endpoint (9411) provides backward compatibility.

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [OpenTelemetry SDK Configuration](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/)
- [Jaeger All-in-One Docker](https://www.jaegertracing.io/docs/latest/getting-started/)
- [Sampling Strategies](https://opentelemetry.io/docs/concepts/sampling/)
