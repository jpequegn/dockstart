/**
 * Express API server that creates background jobs
 *
 * This is the main application entry point. It creates jobs
 * that are processed by the worker sidecar.
 */

const express = require('express');
const { Queue } = require('bullmq');
const IORedis = require('ioredis');

const app = express();
app.use(express.json());

// Redis connection for BullMQ
const connection = new IORedis(process.env.REDIS_URL || 'redis://localhost:6379');

// Create a queue for email jobs
const emailQueue = new Queue('email', { connection });

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

// Create an email job
app.post('/jobs/email', async (req, res) => {
  const { to, subject, body } = req.body;

  const job = await emailQueue.add('send-email', {
    to,
    subject,
    body,
    createdAt: new Date().toISOString()
  });

  console.log(`Created email job ${job.id}`);

  res.json({
    jobId: job.id,
    status: 'queued'
  });
});

// Get job status
app.get('/jobs/:id', async (req, res) => {
  const job = await emailQueue.getJob(req.params.id);

  if (!job) {
    return res.status(404).json({ error: 'Job not found' });
  }

  const state = await job.getState();

  res.json({
    jobId: job.id,
    state,
    data: job.data,
    progress: job.progress
  });
});

const PORT = process.env.PORT || 3000;
app.listen(PORT, () => {
  console.log(`API server listening on port ${PORT}`);
  console.log(`Redis: ${process.env.REDIS_URL || 'redis://localhost:6379'}`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Shutting down...');
  await emailQueue.close();
  await connection.quit();
  process.exit(0);
});
