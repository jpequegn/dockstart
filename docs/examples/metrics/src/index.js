/**
 * Example Express application with Prometheus metrics
 *
 * This demonstrates how to add comprehensive metrics to a Node.js application.
 * Run `dockstart .` in this directory to generate the metrics stack.
 */

const express = require('express');
const client = require('prom-client');

const app = express();
const port = process.env.PORT || 3000;

// =============================================================================
// METRICS SETUP
// =============================================================================

// Enable collection of default metrics (memory, CPU, event loop, etc.)
client.collectDefaultMetrics({
  prefix: 'app_',
  gcDurationBuckets: [0.001, 0.01, 0.1, 1, 2, 5]
});

// Counter: Total HTTP requests
const httpRequestsTotal = new client.Counter({
  name: 'http_requests_total',
  help: 'Total number of HTTP requests',
  labelNames: ['method', 'path', 'status']
});

// Histogram: HTTP request duration
const httpRequestDuration = new client.Histogram({
  name: 'http_request_duration_seconds',
  help: 'Duration of HTTP requests in seconds',
  labelNames: ['method', 'path'],
  buckets: [0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
});

// Gauge: Active connections
const activeConnections = new client.Gauge({
  name: 'active_connections',
  help: 'Number of active HTTP connections'
});

// Counter: Business metric - orders created
const ordersTotal = new client.Counter({
  name: 'orders_total',
  help: 'Total number of orders created',
  labelNames: ['status']
});

// Histogram: Business metric - order processing time
const orderProcessingTime = new client.Histogram({
  name: 'order_processing_seconds',
  help: 'Time to process an order in seconds',
  buckets: [0.1, 0.5, 1, 2, 5, 10]
});

// Gauge: Business metric - pending orders
const pendingOrders = new client.Gauge({
  name: 'pending_orders',
  help: 'Number of orders pending processing'
});

// =============================================================================
// MIDDLEWARE
// =============================================================================

// Metrics middleware - tracks all HTTP requests
app.use((req, res, next) => {
  // Skip metrics endpoint to avoid recursion
  if (req.path === '/metrics') {
    return next();
  }

  const start = process.hrtime.bigint();
  activeConnections.inc();

  res.on('finish', () => {
    const durationNs = process.hrtime.bigint() - start;
    const durationSeconds = Number(durationNs) / 1e9;

    // Normalize path to avoid high cardinality
    const normalizedPath = normalizePath(req.route?.path || req.path);

    httpRequestsTotal.inc({
      method: req.method,
      path: normalizedPath,
      status: res.statusCode
    });

    httpRequestDuration.observe({
      method: req.method,
      path: normalizedPath
    }, durationSeconds);

    activeConnections.dec();
  });

  next();
});

// Normalize paths to prevent high cardinality
function normalizePath(path) {
  // Replace IDs with placeholder
  return path
    .replace(/\/\d+/g, '/:id')
    .replace(/\/[a-f0-9-]{36}/g, '/:uuid');
}

// =============================================================================
// ROUTES
// =============================================================================

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Simulate API endpoints
app.get('/api/users', async (req, res) => {
  // Simulate database query
  await simulateLatency(50, 200);
  res.json({ users: ['alice', 'bob', 'charlie'] });
});

app.get('/api/users/:id', async (req, res) => {
  await simulateLatency(20, 100);
  res.json({ id: req.params.id, name: 'User ' + req.params.id });
});

app.post('/api/orders', express.json(), async (req, res) => {
  const timer = orderProcessingTime.startTimer();
  pendingOrders.inc();

  try {
    // Simulate order processing
    await simulateLatency(100, 500);

    // Randomly fail 10% of orders
    if (Math.random() < 0.1) {
      ordersTotal.inc({ status: 'failed' });
      res.status(500).json({ error: 'Order processing failed' });
    } else {
      ordersTotal.inc({ status: 'success' });
      res.json({ orderId: Date.now(), status: 'created' });
    }
  } finally {
    timer();
    pendingOrders.dec();
  }
});

// Simulate slow endpoint
app.get('/api/reports', async (req, res) => {
  await simulateLatency(1000, 3000);
  res.json({ report: 'generated', items: 1000 });
});

// Simulate error endpoint
app.get('/api/error', (req, res) => {
  res.status(500).json({ error: 'Internal server error' });
});

// =============================================================================
// METRICS ENDPOINT
// =============================================================================

app.get('/metrics', async (req, res) => {
  try {
    res.set('Content-Type', client.register.contentType);
    const metrics = await client.register.metrics();
    res.end(metrics);
  } catch (err) {
    res.status(500).end(err.message);
  }
});

// =============================================================================
// HELPERS
// =============================================================================

function simulateLatency(minMs, maxMs) {
  const delay = Math.random() * (maxMs - minMs) + minMs;
  return new Promise(resolve => setTimeout(resolve, delay));
}

// =============================================================================
// START SERVER
// =============================================================================

app.listen(port, () => {
  console.log(`Server running at http://localhost:${port}`);
  console.log(`Metrics available at http://localhost:${port}/metrics`);
  console.log('');
  console.log('Try these endpoints:');
  console.log(`  GET  http://localhost:${port}/health`);
  console.log(`  GET  http://localhost:${port}/api/users`);
  console.log(`  GET  http://localhost:${port}/api/users/123`);
  console.log(`  POST http://localhost:${port}/api/orders`);
  console.log(`  GET  http://localhost:${port}/api/reports (slow)`);
  console.log(`  GET  http://localhost:${port}/api/error (500)`);
});
