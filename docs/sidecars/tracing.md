# Distributed Tracing Sidecar (Jaeger)

## Overview

When dockstart detects OpenTelemetry or Jaeger client libraries in your project, it automatically generates a **Jaeger distributed tracing backend**. This provides a complete observability solution for development environments without requiring manual configuration.

## What You Get

- **Jaeger All-in-One Container**: A single container running the collector, storage, and UI
- **Auto-Configuration**: Your application receives OpenTelemetry environment variables automatically
- **In-Memory Storage**: Traces are stored in memory (perfect for dev environments)
- **100% Sampling**: All traces are captured in development (no sampling loss)
- **Health Checks**: Docker Compose ensures Jaeger is healthy before starting your app

## Detection

### Supported Libraries by Language

| Language | Libraries | Protocol |
|----------|-----------|----------|
| **Node.js** | `@opentelemetry/sdk-node`<br/>`@opentelemetry/auto-instrumentations-node`<br/>`jaeger-client` | OTLP HTTP (4318) |
| **Go** | `go.opentelemetry.io/otel`<br/>`go.opentelemetry.io/otel/sdk` | OTLP HTTP (4318) |
| **Python** | `opentelemetry-sdk`<br/>`opentelemetry-api`<br/>`opentelemetry-exporter-otlp` | OTLP HTTP (4318) |
| **Rust** | `opentelemetry`<br/>`opentelemetry-sdk`<br/>`tracing-opentelemetry` | OTLP HTTP (4318) |

## Usage

### Automatic Setup

When you run dockstart on a project with tracing libraries:

```bash
$ dockstart ./my-microservice

📂 Analyzing ./my-microservice...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   🔭 Tracing: @opentelemetry/sdk-node (protocol: otlp)
   🔗 Sidecars: [jaeger]

Generated .devcontainer/docker-compose.yml with:
   - app service
   - jaeger service (distributed tracing)

Environment variables will be injected:
   OTEL_SERVICE_NAME=my-microservice
   OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
   OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
   OTEL_TRACES_SAMPLER=always_on
```

### Docker Compose Output

The generated `docker-compose.yml` includes:

```yaml
services:
  app:
    build:
      context: ..
      dockerfile: .devcontainer/Dockerfile
    environment:
      # OpenTelemetry configuration
      - OTEL_SERVICE_NAME=my-microservice
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
      - OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
      - OTEL_TRACES_SAMPLER=always_on
    depends_on:
      jaeger:
        condition: service_healthy

  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "16686:16686" # Jaeger UI
    environment:
      - COLLECTOR_OTLP_ENABLED=true
      - SPAN_STORAGE_TYPE=memory
      - MEMORY_MAX_TRACES=10000
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:16686"]
      interval: 5s
      timeout: 3s
      retries: 3
```

## Environment Variables

### Injected into Your Application

| Variable | Value | Purpose |
|----------|-------|---------|
| `OTEL_SERVICE_NAME` | Project name | Identifies your service in traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `http://jaeger:4318` | Where to send traces |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | Protocol format |
| `OTEL_TRACES_SAMPLER` | `always_on` | Capture 100% of traces |

### Jaeger Configuration

| Variable | Value | Purpose |
|----------|-------|---------|
| `COLLECTOR_OTLP_ENABLED` | `true` | Enable OTLP receiver |
| `SPAN_STORAGE_TYPE` | `memory` | Use in-memory storage |
| `MEMORY_MAX_TRACES` | `10000` | Max traces to keep in memory |

## Accessing Jaeger

### Jaeger UI

Once your development environment is running, access the Jaeger UI:

**URL**: http://localhost:16686

The UI provides:
- **Service List**: View all services sending traces
- **Trace Search**: Find traces by service, operation, tags, or time range
- **Span Waterfall**: Visualize request flow across services
- **Trace Statistics**: View operation latencies and error rates

### Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| `4317` | OTLP gRPC | Receive traces via gRPC |
| `4318` | OTLP HTTP | Receive traces via HTTP |
| `16686` | HTTP | Jaeger UI |

## Finding Traces

### Step 1: Start Your Service

```bash
docker-compose -f .devcontainer/docker-compose.yml up
```

Your app will automatically send traces to Jaeger.

### Step 2: Generate Traffic

Make requests to your application to generate traces:

```bash
# Example for Node.js Express app
curl http://localhost:3000/api/users
curl http://localhost:3000/api/users/123
```

### Step 3: View Traces in Jaeger

1. Open http://localhost:16686
2. In the **Service** dropdown, select your service (e.g., `my-microservice`)
3. Click **Find Traces**
4. View traces in the list
5. Click on a trace to see:
   - **Span waterfall** (timeline of operations)
   - **Span details** (tags, logs, duration)
   - **Service dependencies** (if multi-service)

## OpenTelemetry Setup by Language

### Node.js

Install dependencies:

```bash
npm install \
  @opentelemetry/sdk-node \
  @opentelemetry/api \
  @opentelemetry/auto-instrumentations-node \
  @opentelemetry/sdk-node
```

Initialize at app startup (`index.js`):

```javascript
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');

const sdk = new NodeSDK({
  traceExporter: new OTLPTraceExporter({
    url: process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://localhost:4318/v1/traces',
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();

// Your Express/Fastify/other app initialization follows...
```

### Go

Install dependencies:

```bash
go get go.opentelemetry.io/otel \
  go.opentelemetry.io/otel/sdk \
  go.opentelemetry.io/otel/exporter/otlp/otlptrace/otlptracehttp
```

Initialize at app startup (`main.go`):

```go
package main

import (
    "context"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporter/otlp/otlptrace/otlptracehttp"
)

func init() {
    ctx := context.Background()
    exporter, _ := otlptracehttp.New(ctx)
    
    res, _ := resource.New(ctx,
        resource.WithFromEnv(),
        resource.WithTelemetrySDKVersion(),
    )
    
    provider := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithBatcher(exporter),
    )
    
    otel.SetTracerProvider(provider)
}
```

### Python

Install dependencies:

```bash
pip install \
  opentelemetry-sdk \
  opentelemetry-api \
  opentelemetry-exporter-otlp
```

Initialize at app startup:

```python
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry import trace

exporter = OTLPSpanExporter()
trace.set_tracer_provider(TracerProvider())
trace.get_tracer_provider().add_span_processor(SimpleSpanProcessor(exporter))
```

### Rust

Add dependencies to `Cargo.toml`:

```toml
[dependencies]
opentelemetry = "0.20"
opentelemetry-otlp = "0.13"
opentelemetry-sdk = "0.20"
tracing = "0.1"
tracing-opentelemetry = "0.21"
```

Initialize at app startup:

```rust
use opentelemetry_otlp::new_pipeline;
use tracing_opentelemetry::OpenTelemetryLayer;

let tracer = new_pipeline()
    .tracing()
    .install_simple()
    .unwrap();

let telemetry = OpenTelemetryLayer::new(tracer);
// Add to tracing subscriber...
```

## Analyzing Traces

### Understanding the Span Waterfall

The span waterfall shows:
- **Timeline**: Left to right indicates progression of time
- **Nested spans**: Indentation shows parent-child relationships
- **Duration**: Bar width indicates span duration
- **Service boundaries**: Color changes indicate different services

### Common Operations

1. **Find slow requests**:
   - Click a trace's total duration to sort by slowest
   - Look for spans with large bars

2. **Find errors**:
   - Look for spans with red bars (errors)
   - Click the span to see error details and stack traces

3. **Trace a request across services** (microservices):
   - View the span waterfall across multiple services
   - See where time is spent in each service

## Multi-Service Tracing

If you have multiple services in docker-compose, each with tracing enabled:

```yaml
services:
  api:
    environment:
      - OTEL_SERVICE_NAME=api-service
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
  
  backend:
    environment:
      - OTEL_SERVICE_NAME=backend-service
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
  
  database:
    environment:
      - OTEL_SERVICE_NAME=database-service
      - OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
```

Jaeger will automatically:
- Collect traces from all services
- Show service dependencies
- Correlate related spans across services
- Show distributed trace waterfall

## Troubleshooting

### No traces appearing

1. **Check environment variables**:
   ```bash
   docker-compose exec app env | grep OTEL
   ```

2. **Verify Jaeger is running**:
   ```bash
   curl http://localhost:16686/api/services
   ```

3. **Ensure SDK is initialized** before HTTP server starts:
   - NodeJS: Must call `sdk.start()` before `app.listen()`
   - Go: Initialize tracer provider before creating handlers
   - Python: Initialize before Flask/FastAPI app creation

4. **Check health**:
   ```bash
   docker-compose ps jaeger
   ```

### Traces not appearing in UI

- Make sure service name matches your `OTEL_SERVICE_NAME`
- Generate some traffic: `curl http://localhost:3000/`
- UI updates every few seconds; refresh if needed

### Memory issues

If running many requests, adjust `MEMORY_MAX_TRACES`:

```yaml
jaeger:
  environment:
    - MEMORY_MAX_TRACES=50000  # Increase if needed
```

## Storage Options

For longer data retention, replace in-memory storage:

### With Elasticsearch

```yaml
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:7.10.0
    environment:
      - discovery.type=single-node

  jaeger:
    image: jaegertracing/all-in-one:latest
    environment:
      - SPAN_STORAGE_TYPE=elasticsearch
      - ES_SERVER_URLS=http://elasticsearch:9200
```

### With Cassandra

```yaml
jaeger:
  environment:
    - SPAN_STORAGE_TYPE=cassandra
    - CASSANDRA_SERVERS=cassandra
    - CASSANDRA_KEYSPACE=jaeger_v1_datacenter1
```

## Next Steps

- [View working examples](../examples/tracing/)
- [OpenTelemetry documentation](https://opentelemetry.io/docs/)
- [Jaeger documentation](https://www.jaegertracing.io/docs/)
- [View metrics sidecar documentation](./metrics.md)
