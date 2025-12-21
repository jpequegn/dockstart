# ADR-005: Metrics Stack Sidecar Architecture

**Status**: Accepted
**Date**: 2025-12-21
**Issue**: #57

## Context

Dockstart generates Docker dev environments. To provide better observability in development, we want to add metrics collection and visualization sidecars that automatically instrument and monitor application containers.

### Problem Statement

Developers need to:
1. Monitor application performance metrics during development
2. Visualize trends in resource usage, request latency, error rates
3. Learn production monitoring patterns in a local environment
4. Debug performance issues with real-time metrics

## Research: Metrics Stack Comparison

### Candidates Evaluated

| Feature | Prometheus + Grafana | Datadog | InfluxDB + Telegraf |
|---------|---------------------|---------|---------------------|
| **License** | Open Source | Proprietary | Open Source |
| **Memory Usage** | ~100MB (Prom) + ~50MB (Grafana) | N/A (SaaS) | ~150MB combined |
| **Config Format** | YAML | YAML/API | TOML |
| **Docker Integration** | Excellent | Excellent | Good |
| **Learning Value** | High (K8s standard) | Low | Moderate |
| **Offline Support** | Full | None | Full |
| **Dashboard Ecosystem** | 10,000+ community dashboards | Built-in | Fewer |

### Performance Considerations

Based on development environment requirements:
- **Prometheus**: ~100MB RAM base, scales with metric count
- **Grafana**: ~50MB RAM, minimal CPU when idle
- **Total**: ~150MB RAM for complete stack

### Recommendation: **Prometheus + Grafana**

**Rationale**:
1. **Industry Standard**: Skills transfer directly to Kubernetes/production
2. **Open Source**: No licensing concerns, fully offline-capable
3. **Rich Ecosystem**: 10,000+ pre-built Grafana dashboards
4. **Pull Model**: Apps don't need to know about monitoring
5. **Easy Local Setup**: Well-documented Docker patterns
6. **CNCF Graduated**: Mature, well-maintained project

## Decision

### Primary Choice: Prometheus + Grafana

Use Prometheus for metrics collection and Grafana for visualization as the default metrics stack for dockstart-generated environments.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Docker Compose Network                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌─────────────┐    scrape /metrics     ┌──────────────────────┐   │
│   │             │◄──────────────────────►│                      │   │
│   │     App     │        :3000           │     Prometheus       │   │
│   │             │                        │        :9090         │   │
│   └─────────────┘                        └──────────┬───────────┘   │
│                                                     │               │
│   ┌─────────────┐    scrape /metrics               │               │
│   │             │◄─────────────────────────────────┤               │
│   │   Worker    │        :3001                     │               │
│   │             │                                  │               │
│   └─────────────┘                                  │               │
│                                                    │               │
│   ┌─────────────┐    scrape /metrics               │               │
│   │             │◄─────────────────────────────────┤               │
│   │  Postgres   │        :9187                     │               │
│   │  Exporter   │                                  │               │
│   └─────────────┘                                  │               │
│                                                    │ query          │
│                                                    ▼               │
│                                          ┌──────────────────────┐   │
│                                          │                      │   │
│                                          │      Grafana         │   │
│                                          │        :3001         │   │
│                                          │                      │   │
│                                          └──────────────────────┘   │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Scrape Targets

| Target | Port | Metrics Path | Exporter |
|--------|------|--------------|----------|
| App | 3000 | /metrics | Built-in (prom-client, etc.) |
| Worker | 3001 | /metrics | Built-in |
| Node.js | 9464 | /metrics | prom-client default metrics |
| Postgres | 9187 | /metrics | postgres_exporter |
| Redis | 9121 | /metrics | redis_exporter |
| MySQL | 9104 | /metrics | mysqld_exporter |

## Scrape Interval Analysis

### Research Findings

Based on [Prometheus best practices](https://prometheus.io/docs/prometheus/latest/getting_started/) and [community recommendations](https://www.robustperception.io/keep-it-simple-scrape_interval-id/):

| Environment | Recommended Interval | Rationale |
|-------------|---------------------|-----------|
| Production | 15s | Real-time alerting needs |
| Staging | 15-30s | Near-production behavior |
| Development | 30s | Balance of responsiveness and resource usage |
| CI/Testing | 60s | Minimal overhead |

### Decision: 30s Default for Development

**Rationale**:
1. **Resource Friendly**: Half the data volume of 15s
2. **Still Responsive**: Changes visible within 30 seconds
3. **Matches Grafana Refresh**: Default dashboard refresh is 30s
4. **Stale Threshold**: Well under Prometheus 5-minute staleness

```yaml
global:
  scrape_interval: 30s      # Development-friendly default
  evaluation_interval: 30s  # Align with scrape interval
```

## Grafana Provisioning Strategy

### Decision: File-Based Provisioning

**Rationale**:
1. **Reproducible**: Same dashboards every `docker compose up`
2. **Version Controlled**: Dashboards checked into project
3. **No Manual Setup**: Zero configuration needed
4. **Simpler**: No API tokens or authentication required

### Provisioning Structure

```
.devcontainer/
├── grafana/
│   └── provisioning/
│       ├── datasources/
│       │   └── prometheus.yml      # Auto-configure Prometheus datasource
│       └── dashboards/
│           ├── dashboard.yml       # Dashboard provider config
│           └── app-metrics.json    # Pre-built application dashboard
```

### Datasource Provisioning

```yaml
# prometheus.yml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false
```

### Dashboard Provider

```yaml
# dashboard.yml
apiVersion: 1

providers:
  - name: 'dockstart'
    orgId: 1
    folder: 'Dockstart'
    type: file
    disableDeletion: false
    updateIntervalSeconds: 30
    options:
      path: /etc/grafana/provisioning/dashboards
```

## Dashboard Recommendations Per Language

### Pre-Built Dashboard Strategy

| Language | Dashboard ID | Source | Metrics Required |
|----------|-------------|--------|------------------|
| Node.js | 14058 | Grafana Labs | prom-client default metrics |
| Go | 6671 | Community | promhttp default metrics |
| Python | 15060 | Community | prometheus_client metrics |
| Rust | Custom | Generated | prometheus crate metrics |
| Generic | 1860 | Grafana Labs | Node Exporter metrics |

### Custom Dashboard for Apps Without /metrics

For applications without Prometheus instrumentation:

1. **Container Metrics**: Use cAdvisor or Docker daemon metrics
2. **Database Metrics**: Exporters for Postgres, Redis, MySQL
3. **Basic Health**: HTTP endpoint health checks

```yaml
# Fallback scrape config for apps without /metrics
scrape_configs:
  - job_name: 'app-health'
    static_configs:
      - targets: ['app:3000']
    metrics_path: /health
    scrape_timeout: 5s
```

## Detection Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Analyze Project │────▶│ Detect Metrics   │────▶│ Generate        │
│ Dependencies    │     │ Libraries        │     │ Metrics Stack   │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

### Metrics Library Detection

| Language | Libraries to Detect | Indicates |
|----------|---------------------|-----------|
| Node.js | prom-client, express-prom-bundle | Prometheus-ready |
| Go | prometheus/client_golang, promhttp | Prometheus-ready |
| Python | prometheus-client, prometheus-flask-exporter | Prometheus-ready |
| Rust | prometheus, metrics | Prometheus-ready |
| Any | - | Generate with basic exporters |

### Detection Logic

```go
// Pseudo-code for metrics sidecar detection
func ShouldGenerateMetricsSidecar(detection *Detection) bool {
    // Always generate metrics stack (user can opt-out)
    // Even without app instrumentation, we can still monitor:
    // - Database exporters (Postgres, Redis, MySQL)
    // - Container metrics via cAdvisor
    // - Health endpoints

    return true  // Always beneficial for observability
}

func GetDashboardsForLanguage(language string, libs []string) []string {
    dashboards := []string{"container-metrics.json"}

    if hasPrometheusLib(libs) {
        switch language {
        case "node":
            dashboards = append(dashboards, "nodejs-14058.json")
        case "go":
            dashboards = append(dashboards, "go-6671.json")
        case "python":
            dashboards = append(dashboards, "python-15060.json")
        }
    }

    return dashboards
}
```

## Alertmanager Decision

### Decision: Exclude Alertmanager by Default

**Rationale**:
1. **Overkill for Dev**: Alerts are production-focused
2. **Added Complexity**: Another service to configure
3. **No Recipients**: No on-call rotation in development
4. **Dashboard Sufficient**: Visual monitoring adequate for dev

**Alternative**: Provide as opt-in flag `--with-alerts` for users who want it.

## Configuration Schema

### User-Facing Options (Future)

```yaml
# .dockstart.yml (future config file)
sidecars:
  metrics:
    enabled: true           # Enable metrics stack
    scrape_interval: 30s    # Override default
    dashboards:
      - nodejs             # Include language dashboard
      - postgres           # Include database dashboard
    grafana_port: 3001     # Override default port
    prometheus_port: 9090  # Override default port
```

### Generated Docker Compose

```yaml
services:
  prometheus:
    image: prom/prometheus:v2.51.0
    container_name: prometheus
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--storage.tsdb.retention.time=7d'
      - '--web.enable-lifecycle'
    ports:
      - "9090:9090"
    restart: unless-stopped
    networks:
      - default

  grafana:
    image: grafana/grafana:10.4.0
    container_name: grafana
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
      - GF_AUTH_ANONYMOUS_ENABLED=true
      - GF_AUTH_ANONYMOUS_ORG_ROLE=Viewer
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning:ro
      - grafana-data:/var/lib/grafana
    ports:
      - "3001:3000"
    depends_on:
      - prometheus
    restart: unless-stopped
    networks:
      - default

volumes:
  prometheus-data:
  grafana-data:
```

### Generated prometheus.yml

```yaml
global:
  scrape_interval: 30s
  evaluation_interval: 30s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'app'
    static_configs:
      - targets: ['app:3000']
    metrics_path: /metrics
    scrape_timeout: 10s

  # Conditional: Only if postgres detected
  - job_name: 'postgres'
    static_configs:
      - targets: ['postgres-exporter:9187']
```

## Docker Concepts Reference

### Service Dependencies

```yaml
grafana:
  depends_on:
    prometheus:
      condition: service_healthy
```

### Health Checks

```yaml
prometheus:
  healthcheck:
    test: ["CMD", "wget", "-q", "--spider", "http://localhost:9090/-/healthy"]
    interval: 30s
    timeout: 10s
    retries: 3
```

### Container DNS Resolution

- Services can reference each other by name: `http://prometheus:9090`
- Docker Compose creates a default network for all services
- No need for explicit network configuration in most cases

### Volume Types

| Type | Use Case | Example |
|------|----------|---------|
| Named Volume | Persistent data (metrics, dashboards) | `prometheus-data:/prometheus` |
| Bind Mount | Config files (prometheus.yml, dashboards) | `./prometheus.yml:/etc/prometheus/prometheus.yml:ro` |

## Implementation Plan

### Phase 1: Basic Metrics Stack (Issues #58-60)
- [ ] Detect Prometheus client libraries (#58)
- [ ] Create Prometheus sidecar template (#59)
- [ ] Create Grafana sidecar template (#60)

### Phase 2: Enhanced Features (Issues #61-63)
- [ ] Add database exporters (Postgres, Redis, MySQL) (#61)
- [ ] Create language-specific dashboards (#62)
- [ ] Add opt-in Alertmanager support (#63)

## Consequences

### Positive
- Developers get production-grade observability setup automatically
- Skills transfer directly to Kubernetes/production environments
- Pre-built dashboards reduce time to first insight
- Metrics persist across container restarts

### Negative
- Additional ~150MB RAM usage for Prometheus + Grafana
- Two more services in docker-compose (increased complexity)
- Learning curve for Prometheus query language (PromQL)

### Neutral
- Opinionated choice (Prometheus/Grafana) may not suit all users
- Apps without /metrics still get basic container metrics
- 30s scrape interval trades resolution for resource efficiency

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Prometheus Best Practices - Scrape Interval](https://www.robustperception.io/keep-it-simple-scrape_interval-id/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Grafana Dashboard Provisioning Tutorial](https://grafana.com/tutorials/provision-dashboards-and-data-sources/)
- [Prometheus Client Libraries](https://prometheus.io/docs/instrumenting/clientlibs/)
- [prom-client for Node.js](https://github.com/siimon/prom-client)
- [Node.js Grafana Dashboard (14058)](https://grafana.com/grafana/dashboards/14058-node-js/)
- [Go Grafana Dashboard (6671)](https://grafana.com/grafana/dashboards/6671-go-processes/)
