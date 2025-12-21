# Distributed Tracing Sidecar

Dockstart can generate a distributed tracing sidecar using Jaeger to help you trace requests across your services.

## Overview

When dockstart detects OpenTelemetry libraries in your project, it automatically adds:
- **Jaeger All-in-One**: Trace collection and visualization
- **OTLP Environment Variables**: Pre-configured SDK settings
- **Health Checks**: Ensure tracing is ready before app starts

## Quick Start

### 1. Add OpenTelemetry SDK to Your Project

**Node.js:**
```bash
npm install @opentelemetry/sdk-node @opentelemetry/auto-instrumentations-node
```

**Go:**
```bash
go get go.opentelemetry.io/otel
go get go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp
```

**Python:**
```bash
pip install opentelemetry-sdk opentelemetry-instrumentation
```

**Rust:**
```bash
cargo add opentelemetry opentelemetry-otlp tracing-opentelemetry
```

### 2. Generate Docker Environment

```bash
dockstart
```

### 3. Access Jaeger UI

Open http://localhost:16686 to view traces.

## How It Works

```
┌─────────────┐    OTLP :4317     ┌─────────────┐
│     App     │─────────────────►│   Jaeger    │
└─────────────┘                   │   :16686    │
                                  └─────────────┘
      │                                  ▲
      │ trace context                    │
      ▼ (HTTP headers)                   │
┌─────────────┐    OTLP :4317           │
│   Service   │──────────────────────────┘
│      B      │
└─────────────┘
```

1. Your app sends spans to Jaeger via OTLP (port 4317)
2. Trace context propagates via HTTP headers between services
3. Jaeger collects, stores, and visualizes the traces
4. View in Jaeger UI at http://localhost:16686

## Environment Variables

Dockstart injects these environment variables into your app container:

| Variable | Value | Description |
|----------|-------|-------------|
| `OTEL_SERVICE_NAME` | `${PROJECT_NAME}` | Your service name in traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://jaeger:4317` | Where to send traces |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Sample all traces |
| `OTEL_PROPAGATORS` | `tracecontext,baggage` | W3C standard propagation |

## Instrumenting Your Code

### Node.js (Auto-Instrumentation)

```javascript
// tracing.js - Load this first
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');

const sdk = new NodeSDK({
  autoInstrumentations: getNodeAutoInstrumentations(),
});

sdk.start();
```

```javascript
// package.json
{
  "scripts": {
    "start": "node --require ./tracing.js app.js"
  }
}
```

### Go (Manual Instrumentation)

```go
package main

import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
    "go.opentelemetry.io/otel/sdk/trace"
)

func initTracer() (*trace.TracerProvider, error) {
    exporter, err := otlptracegrpc.New(context.Background())
    if err != nil {
        return nil, err
    }

    tp := trace.NewTracerProvider(
        trace.WithBatcher(exporter),
    )
    otel.SetTracerProvider(tp)
    return tp, nil
}
```

### Python (Auto-Instrumentation)

```bash
# Run with auto-instrumentation
opentelemetry-instrument python app.py
```

Or programmatic setup:

```python
from opentelemetry import trace
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter

provider = TracerProvider()
processor = BatchSpanProcessor(OTLPSpanExporter())
provider.add_span_processor(processor)
trace.set_tracer_provider(provider)
```

## Jaeger UI Features

### Trace Search
- Filter by service, operation, tags, duration
- Time range selection
- Error-only filtering

### Trace View
- Waterfall timeline of spans
- Span details (tags, logs, process info)
- Critical path highlighting

### Service Graph
- Automatic dependency mapping
- Traffic flow visualization
- Error rate indicators

### Trace Comparison
- Compare two traces side-by-side
- Identify performance regressions

## Ports Reference

| Port | Protocol | Description |
|------|----------|-------------|
| 16686 | HTTP | Jaeger UI |
| 4317 | gRPC | OTLP traces (recommended) |
| 4318 | HTTP | OTLP traces (fallback) |
| 6831 | UDP | Jaeger Compact (legacy) |
| 9411 | HTTP | Zipkin compatible |

## Sampling Strategies

### Development (Default)
```yaml
OTEL_TRACES_SAMPLER=parentbased_always_on  # 100% sampling
```

### High-Volume Development
```yaml
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.5  # 50% sampling
```

### Disable Tracing
```yaml
OTEL_TRACES_SAMPLER=always_off
```

## Trace Context Propagation

For traces to connect across services, propagate context via HTTP headers:

### W3C Trace Context (Default)
```
traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01
tracestate: congo=t61rcWkgMzE
```

### B3 Format (Zipkin Compatible)
```
X-B3-TraceId: 80f198ee56343ba864fe8b2a57d3eff7
X-B3-SpanId: e457b5a2e4d86bd1
X-B3-Sampled: 1
```

OpenTelemetry SDKs handle propagation automatically when using:
```yaml
OTEL_PROPAGATORS=tracecontext,baggage,b3multi
```

## Integration with Other Sidecars

### With Metrics (Prometheus + Grafana)
Grafana can visualize Jaeger traces alongside metrics:

```yaml
# Grafana datasource (auto-provisioned)
datasources:
  - name: Jaeger
    type: jaeger
    url: http://jaeger:16686
```

### With Logging (Fluent Bit)
Include trace ID in your logs for correlation:

```javascript
// Node.js example
const { trace } = require('@opentelemetry/api');

function log(message) {
  const span = trace.getActiveSpan();
  const traceId = span?.spanContext().traceId;
  console.log(JSON.stringify({ message, traceId }));
}
```

## Troubleshooting

### No traces appearing

1. **Check Jaeger is healthy:**
   ```bash
   curl http://localhost:16686/api/services
   ```

2. **Verify OTLP endpoint:**
   ```bash
   docker compose logs jaeger | grep OTLP
   ```

3. **Check app environment:**
   ```bash
   docker compose exec app env | grep OTEL
   ```

### Traces missing spans

1. **Ensure context propagation:**
   - Check HTTP client is instrumented
   - Verify headers are passed between services

2. **Check sampling:**
   - Confirm `OTEL_TRACES_SAMPLER` is set correctly
   - Parent spans may control child sampling

### High memory usage

1. **Reduce span volume:**
   ```yaml
   OTEL_TRACES_SAMPLER=parentbased_traceidratio
   OTEL_TRACES_SAMPLER_ARG=0.1  # 10% sampling
   ```

2. **Limit retention:**
   ```yaml
   jaeger:
     command:
       - "--memory.max-traces=10000"
   ```

## Resource Usage

| Component | Memory | CPU (Idle) | CPU (Active) |
|-----------|--------|------------|--------------|
| Jaeger All-in-One | ~100MB | <1% | 2-5% |

## Best Practices

1. **Meaningful span names**: Use operation names, not URLs
   - Good: `user.create`, `order.process`
   - Bad: `POST /api/v1/users`

2. **Add context via tags**:
   ```javascript
   span.setAttribute('user.id', userId);
   span.setAttribute('order.amount', amount);
   ```

3. **Record errors properly**:
   ```javascript
   span.recordException(error);
   span.setStatus({ code: SpanStatusCode.ERROR });
   ```

4. **Use semantic conventions**:
   - `http.method`, `http.url`, `http.status_code`
   - `db.system`, `db.name`, `db.operation`

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
