# Distributed Tracing Quick Start

Get OpenTelemetry tracing working in your dockstart development environment in 5 minutes.

## 1. Check if Your Project Has Tracing Libraries

dockstart automatically detects these tracing libraries:

### Node.js
- `@opentelemetry/sdk-node`
- `@opentelemetry/auto-instrumentations-node`
- `jaeger-client`

### Go
- `go.opentelemetry.io/otel`
- `go.opentelemetry.io/otel/sdk`

### Python
- `opentelemetry-sdk`
- `opentelemetry-api`

### Rust
- `opentelemetry`
- `opentelemetry-sdk`
- `tracing-opentelemetry`

## 2. Generate Development Environment

```bash
dockstart .
```

dockstart will detect tracing libraries and automatically:
- Create `.devcontainer/docker-compose.yml` with Jaeger sidecar
- Generate `.devcontainer/devcontainer.json` with port forwarding
- Set up environment variables in your app service

## 3. Start Jaeger

```bash
docker-compose -f .devcontainer/docker-compose.yml up
```

Wait for output like:
```
jaeger     | {"level":"info","timestamp":"...","msg":"Listening on ..."}
```

## 4. Generate Traces

Make requests to your application:

### Node.js (port 3000)
```bash
curl http://localhost:3000/
```

### Go (port 8080)
```bash
curl http://localhost:8080/
```

### Python (port 8000)
```bash
curl http://localhost:8000/
```

### Rust (port 8080)
```bash
curl http://localhost:8080/
```

## 5. View Traces in Jaeger

1. Open **http://localhost:16686** in your browser
2. In the **Service** dropdown, select your service
3. Click **Find Traces**
4. Click on a trace to see the span waterfall

## What You'll See

The Jaeger UI shows:

```
Timeline of Spans
├── GET / (HTTP request)
│   ├── middleware (Express/FastAPI)
│   └── handler (your code)
└── Database query (if applicable)
```

Each span shows:
- Duration (how long it took)
- Attributes (tags like user_id, http.status_code)
- Logs and events
- Service name and operation name

## Environment Variables

Your app automatically receives:

```bash
OTEL_SERVICE_NAME=my-service           # Your project name
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_TRACES_SAMPLER=always_on          # Capture 100% of traces
```

These are injected by the generated `docker-compose.yml`.

## Add Custom Spans

To trace application-specific operations:

### Node.js
```javascript
const { trace } = require('@opentelemetry/api');
const tracer = trace.getTracer('my-tracer');

const span = tracer.startSpan('my-operation');
try {
  // Do work...
} finally {
  span.end();
}
```

### Go
```go
ctx, span := tracer.Start(context.Background(), "my-operation")
defer span.End()
```

### Python
```python
with tracer.start_as_current_span("my-operation") as span:
    # Do work...
    pass
```

### Rust
```rust
let span = info_span!("my-operation");
let _enter = span.enter();
```

## Troubleshooting

### "No traces found"

1. **Check service name**:
   ```bash
   # The service name in Jaeger UI must match your OTEL_SERVICE_NAME
   docker-compose exec app env | grep OTEL_SERVICE_NAME
   ```

2. **Check Jaeger is running**:
   ```bash
   curl http://localhost:16686/api/services
   ```

3. **Generate more traffic**:
   ```bash
   for i in {1..5}; do curl http://localhost:3000/; sleep 1; done
   ```

4. **Refresh the UI**: Jaeger UI caches. Press Ctrl+Shift+R to hard refresh.

### "Connection refused to Jaeger"

1. Check Jaeger is healthy:
   ```bash
   docker-compose logs jaeger
   docker-compose ps jaeger
   ```

2. Verify environment variable:
   ```bash
   docker-compose exec app env | grep OTEL_EXPORTER_OTLP_ENDPOINT
   ```

3. Ensure app starts AFTER Jaeger is healthy:
   ```bash
   docker-compose down && docker-compose up
   ```

## Next Steps

- [View full tracing documentation](./sidecars/tracing.md)
- [Node.js example](./examples/tracing/nodejs/)
- [Go example](./examples/tracing/go/)
- [Python example](./examples/tracing/python/)
- [Rust example](./examples/tracing/rust/)

## Common Tasks

### See which services are sending traces
```bash
curl http://localhost:16686/api/services
```

### Find traces for a specific operation
1. In Jaeger UI, select your service
2. Select operation from "Operation Name" dropdown
3. Click "Find Traces"

### Find slow requests
1. Click on a trace to see the waterfall
2. Look for spans with large bars (longer duration)
3. Click the span to see details

### Find errors
1. In Jaeger UI, select your service
2. In filters, select "Tags" 
3. Add filter: `error=true`
4. Click "Find Traces"

### Analyze a specific request
1. Open Jaeger UI
2. Find the trace
3. Click on it to see:
   - **Timeline**: When each span happened
   - **Span details**: Attributes, logs, duration
   - **Service map**: Which services were involved
