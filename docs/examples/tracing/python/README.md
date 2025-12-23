# Python OpenTelemetry Example

This example demonstrates how to use OpenTelemetry with Python and Jaeger distributed tracing in a dockstart development environment.

## What This Example Shows

- Setting up OpenTelemetry SDK with Python
- Exporting traces to Jaeger via OTLP HTTP
- Integration with FastAPI/Flask
- Custom span creation with attributes
- Using generated devcontainer
- Viewing traces in Jaeger UI

## Project Structure

```
python/
├── requirements.txt
├── main.py              # FastAPI app with OpenTelemetry
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
curl http://localhost:8000/
curl http://localhost:8000/api/users
curl http://localhost:8000/api/users/123
```

## View Traces

1. Open http://localhost:16686 (Jaeger UI)
2. Select service: `tracing-example-python`
3. Click "Find Traces"
4. Click on a trace to see the span waterfall

## Key Files

### requirements.txt

The example includes OpenTelemetry packages:

```
opentelemetry-sdk==1.21.0
opentelemetry-api==1.21.0
opentelemetry-exporter-otlp==1.21.0
opentelemetry-instrumentation==0.42b0
opentelemetry-instrumentation-fastapi==0.42b0
opentelemetry-instrumentation-requests==0.42b0
fastapi==0.104.1
uvicorn==0.24.0
```

### main.py (Instrumentation)

Initialize OpenTelemetry at application startup:

```python
import os
from fastapi import FastAPI
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.exporter.otlp.proto.http.trace_exporter import OTLPSpanExporter
from opentelemetry import trace
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor
from opentelemetry.instrumentation.requests import RequestsInstrumentor
from opentelemetry.sdk.resources import Resource

# Initialize tracer
resource = Resource.create({
    "service.name": os.getenv("OTEL_SERVICE_NAME", "python-service")
})

exporter = OTLPSpanExporter(
    endpoint=os.getenv(
        "OTEL_EXPORTER_OTLP_ENDPOINT",
        "http://localhost:4318/v1/traces"
    )
)

tracer_provider = TracerProvider(resource=resource)
tracer_provider.add_span_processor(BatchSpanProcessor(exporter))
trace.set_tracer_provider(tracer_provider)

# Instrument libraries
FastAPIInstrumentor.instrument_app(app)
RequestsInstrumentor().instrument()

app = FastAPI()
tracer = trace.get_tracer(__name__)

@app.get("/")
def root():
    with tracer.start_as_current_span("handle_root") as span:
        span.set_attribute("http.method", "GET")
        span.set_attribute("http.url", "/")
        return {"message": "Hello World"}

@app.get("/api/users/{user_id}")
def get_user(user_id: int):
    with tracer.start_as_current_span("get_user") as span:
        span.set_attribute("user.id", user_id)
        return {"id": user_id, "name": f"User {user_id}"}

if __name__ == "__main__":
    import uvicorn
    print(f"Sending traces to {os.getenv('OTEL_EXPORTER_OTLP_ENDPOINT')}")
    uvicorn.run(app, host="0.0.0.0", port=8000)
```

## Environment Variables (auto-injected)

When using the generated `.devcontainer/docker-compose.yml`:

```bash
OTEL_SERVICE_NAME=tracing-example-python
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_TRACES_SAMPLER=always_on
```

## Custom Spans

Create custom spans for application-specific operations:

```python
with tracer.start_as_current_span("my-operation") as span:
    # Add attributes
    span.set_attribute("user.id", "123")
    span.set_attribute("operation.type", "query")
    
    # Add events
    span.add_event("operation started")
    
    # Do work...
    result = expensive_operation()
    
    span.add_event("operation completed", {"result": str(result)})
```

## Instrumentation Options

### FastAPI

FastAPI is automatically instrumented with middleware:

```python
from opentelemetry.instrumentation.fastapi import FastAPIInstrumentor

app = FastAPI()
FastAPIInstrumentor.instrument_app(app)
```

### Requests Library

Instrument HTTP client requests:

```python
from opentelemetry.instrumentation.requests import RequestsInstrumentor

RequestsInstrumentor().instrument()

# Now all requests.get(), requests.post() etc. are traced
import requests
requests.get("http://example.com")
```

### Database

For database tracing, install:

```bash
pip install opentelemetry-instrumentation-sqlalchemy
```

Then instrument:

```python
from opentelemetry.instrumentation.sqlalchemy import SQLAlchemyInstrumentor

SQLAlchemyInstrumentor().instrument(engine=engine)
```

## Trace Analysis

### Finding Your Service in Jaeger

1. Go to http://localhost:16686
2. Service dropdown → select `tracing-example-python`
3. Click "Find Traces"

### Understanding the Waterfall

You'll see spans like:
- `GET /api/users/{user_id}` - FastAPI route (auto-instrumented)
- `get_user` - Custom span you created
- Nested database queries if using SQLAlchemy

### Common Questions

**Q: Why no traces appearing?**
- Ensure instrumentation setup before app starts
- Check `OTEL_EXPORTER_OTLP_ENDPOINT` is correct
- Verify Jaeger health: `docker-compose ps jaeger`
- Check logs: `docker-compose logs app`

**Q: How to add more attributes?**
```python
span.set_attribute("key", "value")
span.set_attribute("numeric.value", 42)
```

**Q: How to trace database calls?**
```bash
pip install opentelemetry-instrumentation-sqlalchemy
```

## Testing

To test the example:

```bash
# Terminal 1: Start services
docker-compose -f .devcontainer/docker-compose.yml up

# Terminal 2: Generate traces
for i in {1..10}; do
  curl http://localhost:8000/api/users/$i
  sleep 0.5
done

# Terminal 3: View in Jaeger
open http://localhost:16686  # or your browser
```

## Next Steps

- Add database instrumentation
- Integrate message queue tracing
- View [main tracing documentation](../../sidecars/tracing.md)
