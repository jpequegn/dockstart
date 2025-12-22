# Rust OpenTelemetry Example

This example demonstrates how to use OpenTelemetry with Rust and Jaeger distributed tracing in a dockstart development environment.

## What This Example Shows

- Setting up OpenTelemetry SDK with Rust
- Exporting traces to Jaeger via OTLP HTTP
- Integration with Axum web framework
- Custom span creation with attributes
- Using generated devcontainer
- Viewing traces in Jaeger UI

## Project Structure

```
rust/
├── Cargo.toml
├── src/
│   └── main.rs          # Axum app with OpenTelemetry
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
2. Select service: `tracing-example-rust`
3. Click "Find Traces"
4. Click on a trace to see the span waterfall

## Key Files

### Cargo.toml

The example imports OpenTelemetry packages:

```toml
[dependencies]
opentelemetry = "0.21"
opentelemetry-sdk = "0.21"
opentelemetry-otlp = "0.14"
tracing = "0.1"
tracing-opentelemetry = "0.22"
tracing-subscriber = "0.3"
axum = "0.7"
tokio = { version = "1", features = ["full"] }
```

### src/main.rs (Instrumentation)

Initialize OpenTelemetry at application startup:

```rust
use axum::{
    extract::Path,
    routing::get,
    Json, Router,
};
use opentelemetry::sdk::resource::Resource;
use opentelemetry::sdk::trace::TracerProvider;
use opentelemetry::sdk::trace::context::TraceState;
use opentelemetry_otlp::new_pipeline;
use std::env;
use tracing::{info, Span};
use tracing_opentelemetry::OpenTelemetryLayer;
use tracing_subscriber::layer::SubscriberExt;

#[tokio::main]
async fn main() {
    // Initialize tracer
    let tracer_provider = new_pipeline()
        .tracing()
        .with_exporter(
            opentelemetry_otlp::new_exporter()
                .http()
                .with_endpoint(
                    env::var("OTEL_EXPORTER_OTLP_ENDPOINT")
                        .unwrap_or_else(|_| "http://localhost:4318/v1/traces".to_string())
                )
                .build()
                .expect("failed to create exporter")
        )
        .with_resource(
            Resource::new(vec![
                opentelemetry::KeyValue::new(
                    "service.name",
                    env::var("OTEL_SERVICE_NAME").unwrap_or_else(|_| "rust-service".to_string())
                ),
            ])
        )
        .install_simple()
        .expect("failed to install tracer");

    // Set up tracing subscriber
    let tracer = tracer_provider.tracer("example-tracer");
    let telemetry = OpenTelemetryLayer::new(tracer);

    tracing_subscriber::registry()
        .with(telemetry)
        .init();

    info!("Server starting on port 8080");
    info!("Sending traces to {}", 
        env::var("OTEL_EXPORTER_OTLP_ENDPOINT")
            .unwrap_or_else(|_| "http://localhost:4318/v1/traces".to_string())
    );

    // Set up routes
    let app = Router::new()
        .route("/", get(root))
        .route("/api/users/:id", get(get_user));

    // Start server
    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080")
        .await
        .unwrap();

    axum::serve(listener, app).await.unwrap();
}

async fn root() -> Json<serde_json::json> {
    Json(serde_json::json!({
        "message": "Hello World"
    }))
}

async fn get_user(Path(id): Path<u32>) -> Json<serde_json::Value> {
    info!(user_id = id, "Getting user");
    
    Json(serde_json::json!({
        "id": id,
        "name": format!("User {}", id)
    }))
}
```

## Environment Variables (auto-injected)

When using the generated `.devcontainer/docker-compose.yml`:

```bash
OTEL_SERVICE_NAME=tracing-example-rust
OTEL_EXPORTER_OTLP_ENDPOINT=http://jaeger:4318
OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf
OTEL_TRACES_SAMPLER=always_on
```

## Custom Spans

Create custom spans using the `tracing` crate:

```rust
use tracing::{info, debug, span, Level};

// Create a span
let span = span!(Level::DEBUG, "my_operation", user_id = 42);
let _enter = span.enter();

info!("Operation started");

// Nested spans
let child_span = span!(Level::DEBUG, "child_operation");
let _child_enter = child_span.enter();

debug!("Child operation");
```

## Automatic Instrumentation

### HTTP Layer

Axum routes are automatically instrumented by `tracing-subscriber`:

```rust
let app = Router::new()
    .route("/", get(root))
    .route("/api/users/:id", get(get_user));
```

Each request generates a span with route information.

### Database Instrumentation

For database operations, use the `tracing-sqlx` crate:

```bash
cargo add tracing-sqlx
```

Then enable in `sqlx` initialization:

```rust
let pool = PgPool::connect(&database_url)
    .instrument(info_span!("database_connect"))
    .await?;
```

### HTTP Client

Instrument `reqwest` client:

```bash
cargo add reqwest --features "tracing"
```

```rust
let client = reqwest::Client::builder()
    .build()?;

client.get("http://example.com")
    .send()
    .await?  // Automatically traced
```

## Trace Analysis

### Finding Your Service in Jaeger

1. Go to http://localhost:16686
2. Service dropdown → select `tracing-example-rust`
3. Click "Find Traces"

### Understanding the Waterfall

You'll see spans like:
- `GET /api/users/:id` - HTTP request
- `Getting user` - Custom log event
- Nested spans for sub-operations

### Common Questions

**Q: Why no traces appearing?**
- Ensure tracer initialization happens before creating Router
- Check `OTEL_EXPORTER_OTLP_ENDPOINT` environment variable
- Verify Jaeger health: `docker-compose ps jaeger`
- Check if `info!` or `debug!` logs are being output

**Q: How to add structured fields to spans?**
```rust
info!(user_id = 123, operation = "create", "User created");
```

**Q: How to create explicit spans?**
```rust
use tracing::info_span;

let span = info_span!("my_operation", user_id = 42);
let _enter = span.enter();
```

## Performance Tips

- Use `BatchSpanProcessor` for better performance (default)
- Set `OTEL_TRACES_SAMPLER=parentbased_traceidratio` for probabilistic sampling in high-traffic scenarios
- Increase `MEMORY_MAX_TRACES` if needed: Docker Compose environment

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

## Troubleshooting

### Build Issues

If you get compilation errors related to OpenTelemetry:

```bash
cargo update
cargo clean
cargo build
```

### No Traces

1. Check if Jaeger is healthy: `docker-compose logs jaeger`
2. Verify endpoint: `echo $OTEL_EXPORTER_OTLP_ENDPOINT`
3. Check app logs: `docker-compose logs app`

### Slow Startup

OpenTelemetry initialization can take a moment. Wait for the "Server starting" message.

## Next Steps

- Add database instrumentation with `sqlx` tracing
- Integrate message queue tracing
- View [main tracing documentation](../../sidecars/tracing.md)
