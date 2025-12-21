# Metrics Stack Sidecar (Prometheus + Grafana)

dockstart automatically generates a complete metrics observability stack when it detects Prometheus client libraries in your project.

## Overview

The metrics stack includes:

- **Prometheus**: Time-series database that scrapes metrics from your application
- **Grafana**: Visualization platform with pre-configured dashboards
- **Database Exporters**: Optional exporters for PostgreSQL and Redis metrics

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     App     │     │ Prometheus  │     │   Grafana   │
│  /metrics   │────▶│  (scrapes)  │────▶│ (dashboards)│
└─────────────┘     └─────────────┘     └─────────────┘
       ▲                   │
       │              every 15s
       │
┌──────┴──────┐
│   Worker    │  (if detected)
│  /metrics   │
└─────────────┘
```

## Detected Libraries

dockstart detects the following Prometheus client libraries:

| Language | Libraries | Default Port |
|----------|-----------|--------------|
| Node.js | `prom-client`, `express-prometheus-middleware` | 3000 |
| Go | `prometheus/client_golang`, `prometheus/promhttp` | 8080 |
| Python | `prometheus-client`, `prometheus-fastapi-instrumentator` | 8000 |
| Rust | `prometheus`, `metrics` | 8080 |

## Generated Files

When metrics libraries are detected, dockstart generates:

```
.devcontainer/
├── docker-compose.yml          # Includes prometheus + grafana services
├── prometheus/
│   └── prometheus.yml          # Prometheus scrape configuration
└── grafana/
    └── provisioning/
        ├── datasources/
        │   └── prometheus.yml  # Auto-configured Prometheus datasource
        └── dashboards/
            ├── provider.yml    # Dashboard auto-discovery config
            └── app-metrics.json # Pre-built application dashboard
```

## Port Allocation

| Service | Internal Port | External Port | URL |
|---------|---------------|---------------|-----|
| Prometheus | 9090 | 9090 | http://localhost:9090 |
| Grafana | 3000 | 3001 | http://localhost:3001 |
| PostgreSQL Exporter | 9187 | 9187 | (if Postgres detected) |
| Redis Exporter | 9121 | 9121 | (if Redis detected) |

## Configuration

### Prometheus Configuration

The generated `prometheus.yml` includes:

```yaml
global:
  scrape_interval: 30s      # Global default
  evaluation_interval: 30s

scrape_configs:
  # Prometheus self-monitoring
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  # Your application
  - job_name: 'myapp'
    static_configs:
      - targets: ['app:3000']
    metrics_path: /metrics
    scrape_interval: 15s    # More frequent for app

  # Worker (if detected)
  - job_name: 'myapp-worker'
    static_configs:
      - targets: ['worker:3000']
    scrape_interval: 30s

  # PostgreSQL (if detected)
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']

  # Redis (if detected)
  - job_name: 'redis'
    static_configs:
      - targets: ['redis-exporter:9121']
```

### Grafana Configuration

Grafana is pre-configured with:

- **Prometheus datasource**: Auto-connected to Prometheus container
- **Anonymous access**: Enabled for easy development (Viewer role)
- **Admin credentials**: `admin` / `admin` (change in production!)
- **Pre-built dashboard**: Application metrics with 4 panels

## Pre-built Dashboard

The generated dashboard includes:

### Request Rate Panel
Shows HTTP requests per second, grouped by method and path.

```promql
rate(http_requests_total{job="myapp"}[5m])
```

### Response Time Percentiles Panel
Shows p50, p95, and p99 latency distribution.

```promql
histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{job="myapp"}[5m])) by (le))
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{job="myapp"}[5m])) by (le))
histogram_quantile(0.50, sum(rate(http_request_duration_seconds_bucket{job="myapp"}[5m])) by (le))
```

### Error Rate Panel
Shows the percentage of 5xx responses over time.

```promql
sum(rate(http_requests_total{job="myapp",status=~"5.."}[5m])) / sum(rate(http_requests_total{job="myapp"}[5m]))
```

### Requests by Status Code Panel
Shows request breakdown by HTTP status code.

```promql
sum(rate(http_requests_total{job="myapp"}[5m])) by (status)
```

## Adding Custom Metrics

### Node.js (prom-client)

```javascript
const client = require('prom-client');

// Enable default metrics (memory, CPU, etc.)
client.collectDefaultMetrics();

// Custom counter
const httpRequestsTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'path', 'status']
});

// Custom histogram for response times
const httpRequestDuration = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'HTTP request duration in seconds',
  labelNames: ['method', 'path'],
  buckets: [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
});

// Custom gauge for active connections
const activeConnections = new client.Gauge({
  name: 'active_connections',
  help: 'Number of active connections'
});

// Middleware to track metrics
app.use((req, res, next) => {
  const start = Date.now();
  activeConnections.inc();

  res.on('finish', () => {
    const duration = (Date.now() - start) / 1000;
    httpRequestsTotal.inc({
      method: req.method,
      path: req.route?.path || req.path,
      status: res.statusCode
    });
    httpRequestDuration.observe({
      method: req.method,
      path: req.route?.path || req.path
    }, duration);
    activeConnections.dec();
  });

  next();
});

// Expose metrics endpoint
app.get('/metrics', async (req, res) => {
  res.set('Content-Type', client.register.contentType);
  res.end(await client.register.metrics());
});
```

### Go (prometheus/client_golang)

```go
package main

import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "path"},
    )
)

func init() {
    prometheus.MustRegister(httpRequestsTotal)
    prometheus.MustRegister(httpRequestDuration)
}

func metricsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, r.URL.Path))
        defer timer.ObserveDuration()

        // Wrap response writer to capture status code
        wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
        next.ServeHTTP(wrapped, r)

        httpRequestsTotal.WithLabelValues(r.Method, r.URL.Path,
            fmt.Sprintf("%d", wrapped.statusCode)).Inc()
    })
}

func main() {
    http.Handle("/metrics", promhttp.Handler())
    http.Handle("/", metricsMiddleware(yourHandler))
    http.ListenAndServe(":8080", nil)
}
```

### Python (prometheus-client)

```python
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
import time

# Define metrics
http_requests_total = Counter(
    'http_requests_total',
    'Total HTTP requests',
    ['method', 'path', 'status']
)

http_request_duration = Histogram(
    'http_request_duration_seconds',
    'HTTP request duration in seconds',
    ['method', 'path'],
    buckets=[.01, .05, .1, .25, .5, 1, 2.5, 5, 10]
)

# FastAPI example
from fastapi import FastAPI, Request, Response

app = FastAPI()

@app.middleware("http")
async def metrics_middleware(request: Request, call_next):
    start_time = time.time()
    response = await call_next(request)
    duration = time.time() - start_time

    http_requests_total.labels(
        method=request.method,
        path=request.url.path,
        status=response.status_code
    ).inc()

    http_request_duration.labels(
        method=request.method,
        path=request.url.path
    ).observe(duration)

    return response

@app.get("/metrics")
def metrics():
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)
```

## Customizing Dashboards

### Creating Custom Panels

1. Open Grafana at http://localhost:3001
2. Navigate to the pre-built dashboard
3. Click "Add panel"
4. Enter your PromQL query
5. Configure visualization options
6. Save the dashboard

### Exporting Dashboards

To export your customized dashboard:

1. Click the gear icon (Dashboard settings)
2. Select "JSON Model"
3. Copy the JSON and save to `.devcontainer/grafana/provisioning/dashboards/`

### Common PromQL Patterns

```promql
# Request rate per second
rate(http_requests_total[5m])

# Average response time
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# 95th percentile response time
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error percentage
100 * sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# Top 5 slowest endpoints
topk(5, avg by (path) (rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])))

# Memory usage (Go default metrics)
process_resident_memory_bytes

# Active goroutines (Go)
go_goroutines
```

## Database Exporters

### PostgreSQL Exporter

When PostgreSQL is detected, the `postgres-exporter` is automatically added:

```yaml
postgres-exporter:
  image: quay.io/prometheuscommunity/postgres-exporter:latest
  environment:
    - DATA_SOURCE_NAME=postgresql://postgres:postgres@postgres:5432/myapp_dev?sslmode=disable
```

Key metrics:
- `pg_stat_database_tup_fetched`: Rows fetched
- `pg_stat_database_tup_inserted`: Rows inserted
- `pg_stat_database_numbackends`: Active connections
- `pg_stat_database_blks_read`: Disk blocks read

### Redis Exporter

When Redis is detected, the `redis-exporter` is automatically added:

```yaml
redis-exporter:
  image: oliver006/redis_exporter:latest
  environment:
    - REDIS_ADDR=redis://redis:6379
```

Key metrics:
- `redis_connected_clients`: Number of clients
- `redis_used_memory_bytes`: Memory usage
- `redis_commands_processed_total`: Commands executed
- `redis_keyspace_hits_total`: Cache hit rate

## Troubleshooting

### Metrics Not Appearing

1. **Check endpoint**: Verify `/metrics` returns Prometheus format
   ```bash
   curl http://localhost:3000/metrics
   ```

2. **Check Prometheus targets**: Visit http://localhost:9090/targets
   - All targets should show "UP" state
   - Check for connection errors

3. **Check container networking**:
   ```bash
   docker compose logs prometheus
   ```

### Grafana Dashboard Empty

1. **Verify datasource**: Go to Configuration > Data Sources > Prometheus > Test

2. **Check time range**: Ensure the time picker includes recent data

3. **Verify metric names**: Check Prometheus UI (http://localhost:9090/graph) for available metrics

### High Cardinality Warnings

Avoid adding high-cardinality labels (like user IDs) to metrics. Use:
- HTTP method (limited values)
- Route patterns (not full URLs)
- Status code categories (2xx, 4xx, 5xx)

## Best Practices

1. **Use consistent metric names**: Follow Prometheus naming conventions
   - `http_requests_total` (counter)
   - `http_request_duration_seconds` (histogram)
   - `active_connections` (gauge)

2. **Choose appropriate label cardinality**: Keep label combinations under 1000

3. **Set reasonable scrape intervals**: 15s for applications, 30s for databases

4. **Use histograms for latency**: They provide percentiles without pre-aggregation

5. **Add help text**: Every metric should have a descriptive help string

## See Also

- [PromQL Cheatsheet](./promql-cheatsheet.md)
- [Dashboard Customization Guide](./dashboard-customization.md)
- [Example Project](../examples/metrics/)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
