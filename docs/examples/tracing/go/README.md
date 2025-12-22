# Go OpenTelemetry Example

This example demonstrates how to use OpenTelemetry with Go and Jaeger distributed tracing in a dockstart development environment.

## What This Example Shows

- Setting up OpenTelemetry SDK with Go
- Exporting traces to Jaeger via OTLP HTTP
- Custom span creation with attributes
- Using generated devcontainer
- Viewing traces in Jaeger UI

## Project Structure

```
go/
├── go.mod
├── go.sum
├── main.go              # HTTP server with OpenTelemetry
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
curl http://localhost:8080/
curl http://localhost:8080/api/users
curl http://localhost:8080/api/users/123
```

## View Traces

1. Open http://localhost:16686 (Jaeger UI)
2. Select service: `tracing-example-go`
3. Click "Find Traces"
4. Click on a trace to see the span waterfall

## Key Files

### go.mod

The example imports OpenTelemetry packages:

```go
require (
    go.opentelemetry.io/otel v1.21.0
    go.opentelemetry.io/otel/sdk v1.21.0
    go.opentelemetry.io/otel/exporter/otlp/otlptrace/otlptracehttp v1.21.0
)
```

### main.go (Instrumentation)

Initialize OpenTelemetry at application startup:

```go
package main

import (
    "context"
    "log"
    "net/http"
    "os"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/resource"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    "go.opentelemetry.io/otel/exporter/otlp/otlptrace/otlptracehttp"
    semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

func initTracer() error {
    ctx := context.Background()
    
    exporter, err := otlptracehttp.New(ctx)
    if err != nil {
        return err
    }
    
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String(os.Getenv("OTEL_SERVICE_NAME")),
        ),
    )
    if err != nil {
        return err
    }
    
    tp := sdktrace.NewTracerProvider(
        sdktrace.WithResource(res),
        sdktrace.WithBatcher(exporter),
    )
    
    otel.SetTracerProvider(tp)
    return nil
}

func main() {
    if err := initTracer(); err != nil {
        log.Fatalf("failed to init tracer: %v", err)
    }
    
    tracer := otel.Tracer("example-tracer")
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        ctx, span := tracer.Start(r.Context(), "handle_root")
        defer span.End()
        
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{"message":"Hello World"}`))
    })
    
    log.Println("Server running on port 8080")
    log.Printf("Sending traces to %s\n", os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
    http.ListenAndServe(":8080", nil)
}
```

## Environment Variables (auto-injected)

When using the generated `.devcontainer/docker-compose.yml`:

```bash
OTEL_SERVICE_NAME=tracing-example-go
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_TRACES_SAMPLER=always_on
```

## Custom Spans

Create custom spans for application-specific operations:

```go
ctx, span := tracer.Start(context.Background(), "my-operation")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("user.id", "123"),
    attribute.Int("items.count", 42),
)

// Add events
span.AddEvent("operation started")

// Do work...
span.AddEvent("operation completed")
```

## Trace Analysis

### Finding Your Service in Jaeger

1. Go to http://localhost:16686
2. Service dropdown → select `tracing-example-go`
3. Click "Find Traces"

### Understanding the Waterfall

You'll see spans like:
- `HTTP GET /` - HTTP request handler
- `handle_root` - Custom span you created
- Nested spans if you make calls to other services

### Common Questions

**Q: Why no traces appearing?**
- Ensure `initTracer()` is called before HTTP server starts
- Check `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable
- Verify Jaeger health: `docker-compose ps jaeger`

**Q: How to add database tracing?**
```bash
go get go.opentelemetry.io/otel/instrumentation/database/sql
```

**Q: How to trace gRPC calls?**
```bash
go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
```

## Testing

To test the example:

```bash
# Terminal 1: Start services
docker-compose -f .devcontainer/docker-compose.yml up

# Terminal 2: Generate traces
for i in {1..10}; do
  curl http://localhost:8080/api/users/$i
  sleep 0.5
done

# Terminal 3: View in Jaeger
open http://localhost:16686  # or your browser
```

## Next Steps

- Add middleware for automatic HTTP instrumentation
- Integrate database connection tracing
- View [main tracing documentation](../../sidecars/tracing.md)
