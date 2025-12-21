# PromQL Cheatsheet

Quick reference for Prometheus Query Language (PromQL) commonly used with dockstart metrics.

## Basic Selectors

```promql
# Exact match
http_requests_total{method="GET"}

# Regex match
http_requests_total{path=~"/api/.*"}

# Regex not match
http_requests_total{status!~"2.."}

# Multiple labels
http_requests_total{method="POST", path="/api/users"}
```

## Functions

### Rate and Increase

```promql
# Per-second rate (for counters)
rate(http_requests_total[5m])

# Total increase over time
increase(http_requests_total[1h])

# Rate adjusted for resets
irate(http_requests_total[5m])  # Instant rate (last 2 points)
```

### Aggregation

```promql
# Sum all series
sum(rate(http_requests_total[5m]))

# Sum by label
sum by (method) (rate(http_requests_total[5m]))

# Sum without label
sum without (instance) (rate(http_requests_total[5m]))

# Average
avg(http_request_duration_seconds)

# Count of series
count(http_requests_total)

# Min/Max
min by (path) (http_request_duration_seconds)
max by (path) (http_request_duration_seconds)
```

### Histogram Functions

```promql
# Percentiles (p50, p90, p95, p99)
histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.90, rate(http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Percentile grouped by label
histogram_quantile(0.95, sum by (path, le) (rate(http_request_duration_seconds_bucket[5m])))

# Average from histogram
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])
```

## HTTP Metrics Patterns

### Request Rate

```promql
# Total requests per second
sum(rate(http_requests_total[5m]))

# Requests by method
sum by (method) (rate(http_requests_total[5m]))

# Requests by path
sum by (path) (rate(http_requests_total[5m]))

# Requests by status code
sum by (status) (rate(http_requests_total[5m]))

# Top 5 busiest endpoints
topk(5, sum by (path) (rate(http_requests_total[5m])))
```

### Error Rate

```promql
# Error rate percentage (5xx)
100 * sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))

# Error rate by endpoint
100 * sum by (path) (rate(http_requests_total{status=~"5.."}[5m])) / sum by (path) (rate(http_requests_total[5m]))

# Client errors (4xx)
sum(rate(http_requests_total{status=~"4.."}[5m]))

# All non-2xx responses
sum(rate(http_requests_total{status!~"2.."}[5m]))
```

### Latency

```promql
# Average response time
rate(http_request_duration_seconds_sum[5m]) / rate(http_request_duration_seconds_count[5m])

# Average by endpoint
sum by (path) (rate(http_request_duration_seconds_sum[5m])) / sum by (path) (rate(http_request_duration_seconds_count[5m]))

# 95th percentile overall
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le))

# 95th percentile by endpoint
histogram_quantile(0.95, sum by (path, le) (rate(http_request_duration_seconds_bucket[5m])))

# Slowest endpoints (by p95)
topk(5, histogram_quantile(0.95, sum by (path, le) (rate(http_request_duration_seconds_bucket[5m]))))
```

### Throughput

```promql
# Requests per minute
sum(rate(http_requests_total[1m])) * 60

# Requests per hour
sum(increase(http_requests_total[1h]))

# Concurrent requests (if using gauge)
avg_over_time(active_connections[5m])
```

## Database Metrics

### PostgreSQL

```promql
# Active connections
pg_stat_database_numbackends

# Rows fetched per second
rate(pg_stat_database_tup_fetched[5m])

# Rows inserted per second
rate(pg_stat_database_tup_inserted[5m])

# Cache hit ratio
pg_stat_database_blks_hit / (pg_stat_database_blks_hit + pg_stat_database_blks_read)

# Transaction rate
rate(pg_stat_database_xact_commit[5m]) + rate(pg_stat_database_xact_rollback[5m])
```

### Redis

```promql
# Connected clients
redis_connected_clients

# Memory usage
redis_used_memory_bytes / 1024 / 1024  # MB

# Commands per second
rate(redis_commands_processed_total[5m])

# Cache hit rate
redis_keyspace_hits_total / (redis_keyspace_hits_total + redis_keyspace_misses_total)

# Keys per database
redis_db_keys{db="db0"}
```

## System Metrics

### Memory

```promql
# Memory usage (Go apps)
process_resident_memory_bytes / 1024 / 1024  # MB

# Heap usage (Node.js with default metrics)
nodejs_heap_size_used_bytes / nodejs_heap_size_total_bytes

# Memory usage percentage
100 * process_resident_memory_bytes / node_memory_MemTotal_bytes
```

### CPU

```promql
# CPU usage rate
rate(process_cpu_seconds_total[5m])

# CPU usage percentage (if using node_exporter)
100 - (avg by(instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

### Event Loop (Node.js)

```promql
# Event loop lag
nodejs_eventloop_lag_seconds

# Active handles
nodejs_active_handles_total

# Active requests
nodejs_active_requests_total
```

## Time Functions

```promql
# Current timestamp
time()

# Days since epoch
time() / 86400

# Time since last value change
time() - timestamp(my_metric)

# Filter by time
http_requests_total offset 1h  # 1 hour ago
http_requests_total @ 1609459200  # At Unix timestamp
```

## Operators

### Arithmetic

```promql
# Addition/subtraction
metric_a + metric_b
metric_a - metric_b

# Multiplication/division
metric_a * 100
metric_a / 1024

# Modulo
metric_a % 60
```

### Comparison (returns 0 or 1)

```promql
# Greater than
http_request_duration_seconds > 1

# Greater than or equal
error_rate >= 0.05

# Equal
status == 200

# Not equal
status != 500
```

### Logical

```promql
# AND (both must exist)
metric_a and metric_b

# OR (union)
metric_a or metric_b

# UNLESS (exclude matching)
metric_a unless metric_b
```

## Labels

```promql
# Add label
my_metric * on() group_left label_replace(my_metric, "new_label", "value", "", "")

# Rename label
label_replace(my_metric, "new_name", "$1", "old_name", "(.*)")

# Drop label
sum without (label_to_drop) (my_metric)

# Keep only specific labels
sum by (label_to_keep) (my_metric)
```

## Subqueries

```promql
# Rate of rate (acceleration)
rate(rate(http_requests_total[5m])[30m:1m])

# Max rate over time
max_over_time(rate(http_requests_total[5m])[1h:])

# Smoothed average
avg_over_time(my_metric[5m:1m])
```

## Alerting Patterns

```promql
# High error rate
sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05

# High latency
histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) > 1

# Low request rate (possible outage)
sum(rate(http_requests_total[5m])) < 1

# Instance down
up == 0

# Disk almost full
100 - (node_filesystem_avail_bytes / node_filesystem_size_bytes * 100) > 90
```

## Recording Rules Examples

For frequently used queries, create recording rules:

```yaml
# prometheus.yml or rules.yml
groups:
  - name: http_metrics
    rules:
      - record: job:http_requests:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))

      - record: job:http_errors:rate5m
        expr: sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))

      - record: job:http_error_percentage
        expr: 100 * job:http_errors:rate5m / job:http_requests:rate5m

      - record: job:http_latency_p95
        expr: histogram_quantile(0.95, sum by (job, le) (rate(http_request_duration_seconds_bucket[5m])))
```

## Common Gotchas

### Counter Resets

```promql
# Use rate() or increase(), not raw counter values
rate(http_requests_total[5m])  # Correct
http_requests_total            # Wrong - shows cumulative total
```

### Range Vector vs Instant Vector

```promql
# Range vector (with [duration])
http_requests_total[5m]        # Returns matrix

# Instant vector (without duration)
http_requests_total            # Returns vector

# Functions that need range vectors
rate(http_requests_total[5m])  # Correct
rate(http_requests_total)      # Error
```

### Label Matching

```promql
# Binary operations require matching labels
metric_a + metric_b            # Only works if labels match
metric_a + ignoring(extra_label) metric_b  # Ignore specific label
metric_a + on(common_label) metric_b       # Match only on specific label
```

## See Also

- [Metrics Stack Documentation](./metrics.md)
- [Dashboard Customization Guide](./dashboard-customization.md)
- [Prometheus Query Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Prometheus Query Functions](https://prometheus.io/docs/prometheus/latest/querying/functions/)
