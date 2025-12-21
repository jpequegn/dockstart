/**
 * BullMQ Worker
 *
 * This file is the entry point for the worker sidecar.
 * It processes jobs created by the main application.
 *
 * Run with: npm run worker
 * Or: node src/worker.js
 */

const { Worker } = require('bullmq');
const IORedis = require('ioredis');

// Import job processors
const { processEmail } = require('./jobs/email');

// Redis connection
const connection = new IORedis(process.env.REDIS_URL || 'redis://localhost:6379');

// Concurrency from environment or default to 2
const concurrency = parseInt(process.env.WORKER_CONCURRENCY || '2', 10);

console.log(`Starting worker with concurrency: ${concurrency}`);
console.log(`Redis: ${process.env.REDIS_URL || 'redis://localhost:6379'}`);

// Create the worker
const worker = new Worker('email', async (job) => {
  console.log(`Processing job ${job.id}: ${job.name}`);

  switch (job.name) {
    case 'send-email':
      return await processEmail(job);
    default:
      throw new Error(`Unknown job type: ${job.name}`);
  }
}, {
  connection,
  concurrency
});

// Event handlers
worker.on('completed', (job, result) => {
  console.log(`Job ${job.id} completed:`, result);
});

worker.on('failed', (job, err) => {
  console.error(`Job ${job.id} failed:`, err.message);
});

worker.on('error', (err) => {
  console.error('Worker error:', err);
});

console.log('Worker started, waiting for jobs...');

// Graceful shutdown
process.on('SIGTERM', async () => {
  console.log('Received SIGTERM, shutting down gracefully...');
  await worker.close();
  await connection.quit();
  console.log('Worker shut down complete');
  process.exit(0);
});

process.on('SIGINT', async () => {
  console.log('Received SIGINT, shutting down gracefully...');
  await worker.close();
  await connection.quit();
  process.exit(0);
});
