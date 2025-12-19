const express = require('express');
const pino = require('pino');
const pinoHttp = require('pino-http');

// Create pino logger with JSON output
const logger = pino({
  level: process.env.LOG_LEVEL || 'info',
  formatters: {
    level: (label) => ({ level: label }),
  },
});

const app = express();
const port = process.env.PORT || 3000;

// HTTP request logging middleware
app.use(pinoHttp({ logger }));

// JSON body parser
app.use(express.json());

// Routes
app.get('/', (req, res) => {
  logger.info({ path: '/' }, 'Root endpoint accessed');
  res.json({ status: 'ok', message: 'Log Aggregator Example' });
});

app.get('/health', (req, res) => {
  logger.debug({ path: '/health' }, 'Health check');
  res.json({ status: 'healthy' });
});

app.post('/users', (req, res) => {
  const { name, email } = req.body;

  if (!name || !email) {
    logger.warn({ body: req.body }, 'Invalid user data received');
    return res.status(400).json({ error: 'Name and email required' });
  }

  // Simulate user creation
  const userId = Math.random().toString(36).substring(7);
  logger.info({ userId, name, email }, 'User created successfully');

  res.status(201).json({ id: userId, name, email });
});

app.get('/error', (req, res) => {
  logger.error({ path: '/error' }, 'Intentional error endpoint');
  res.status(500).json({ error: 'Intentional error for testing' });
});

// Start server
app.listen(port, () => {
  logger.info({ port }, 'Server started');
});
