# Node.js OpenTelemetry Example

This example demonstrates how to use OpenTelemetry with Node.js and Jaeger distributed tracing in a dockstart development environment.

## What This Example Shows

- Setting up OpenTelemetry SDK with Node.js
- Auto-instrumentation for Express
- Custom span creation
- Accessing generated devcontainer
- Viewing traces in Jaeger UI

## Project Structure

```
nodejs/
├── package.json
├── index.js              # Express app with OpenTelemetry
├── Dockerfile
├── .devcontainer/
│   ├── devcontainer.json
│   ├── Dockerfile
│   └── docker-compose.yml
└── README.md
```

## Setup

### 1. Generate devcontainer (automated by dockstart)

```bash
dockstart .
```

This generates `.devcontainer/` with Jaeger included.

### 2. Start development environment

```bash
docker-compose -f .devcontainer/docker-compose.yml up
```

### 3. Make requests to generate traces

```bash
curl http://localhost:3000/
curl http://localhost:3000/api/users
curl http://localhost:3000/api/users/123
```

## View Traces

1. Open http://localhost:16686 (Jaeger UI)
2. Select service: `tracing-example-nodejs`
3. Click "Find Traces"
4. Click on a trace to see the span waterfall

## Key Files

### package.json

The example includes OpenTelemetry packages:

```json
{
  "dependencies": {
    "@opentelemetry/sdk-node": "^0.45.0",
    "@opentelemetry/api": "^1.7.0",
    "@opentelemetry/auto-instrumentations-node": "^0.43.0",
    "@opentelemetry/exporter-trace-otlp-http": "^0.45.0",
    "express": "^4.18.2"
  }
}
```

### index.js (Instrumentation)

The OpenTelemetry SDK is initialized at the very start:

```javascript
const { NodeSDK } = require('@opentelemetry/sdk-node');
const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-http');
const { BatchSpanProcessor } = require('@opentelemetry/sdk-trace-node');

const sdk = new NodeSDK({
  traceExporter: new OTLPTraceExporter({
    url: process.env.OTEL_EXPORTER_OTLP_ENDPOINT || 'http://localhost:4318/v1/traces',
  }),
  instrumentations: [getNodeAutoInstrumentations()],
});

sdk.start();

// Import after SDK initialization
const express = require('express');
const { trace } = require('@opentelemetry/api');

const app = express();
const tracer = trace.getTracer('example-tracer');

// Express routes...
app.get('/', (req, res) => {
  const span = tracer.startSpan('handle_root');
  res.json({ message: 'Hello World' });
  span.end();
});

app.listen(3000, () => {
  console.log('Server running on port 3000');
  console.log('Sending traces to', process.env.OTEL_EXPORTER_OTLP_ENDPOINT);
});
```

## Environment Variables (auto-injected)

When using the generated `.devcontainer/docker-compose.yml`:

```bash
OTEL_SERVICE_NAME=tracing-example-nodejs
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_TRACES_SAMPLER=always_on
```

## Custom Spans

Create custom spans for application-specific operations:

```javascript
const span = tracer.startSpan('my-operation');
try {
  // Do work...
  span.addEvent('operation started');
  span.setAttributes({ 'user.id': userId });
} finally {
  span.end();
}
```

## Trace Analysis

### Finding Your Service in Jaeger

1. Go to http://localhost:16686
2. Service dropdown → select `tracing-example-nodejs`
3. Click "Find Traces"

### Understanding the Waterfall

You'll see spans like:
- `GET /` - HTTP request (auto-instrumented)
- `middleware` - Express middleware (auto-instrumented)
- `handle_root` - Custom span you created

### Common Questions

**Q: Why no traces appearing?**
- Ensure SDK is initialized before importing express
- Check `OTEL_EXPORTER_OTLP_ENDPOINT` points to Jaeger
- Verify Jaeger health: `docker-compose ps jaeger`

**Q: How to add more custom spans?**
```javascript
const span = tracer.startSpan('operation-name');
// ... do work ...
span.end();
```

**Q: How to see logs in spans?**
```javascript
span.addEvent('operation started', { 'step': '1' });
```

## Next Steps

- Add database instrumentation: `@opentelemetry/instrumentation-pg`
- Add Redis tracing: `@opentelemetry/instrumentation-redis-4`
- View [main tracing documentation](../../sidecars/tracing.md)
