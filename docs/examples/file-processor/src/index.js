const express = require('express');
const multer = require('multer');
const path = require('path');
const fs = require('fs');

const app = express();
const port = process.env.PORT || 3000;

// Environment variables from devcontainer
const UPLOAD_PATH = process.env.UPLOAD_PATH || '/uploads/pending';
const PROCESSED_PATH = process.env.PROCESSED_PATH || '/uploads/processed';
const FAILED_PATH = process.env.FAILED_PATH || '/uploads/failed';

// Ensure directories exist
[UPLOAD_PATH, PROCESSED_PATH, FAILED_PATH].forEach(dir => {
  if (!fs.existsSync(dir)) {
    fs.mkdirSync(dir, { recursive: true });
  }
});

// Configure multer for file uploads
const storage = multer.diskStorage({
  destination: UPLOAD_PATH,
  filename: (req, file, cb) => {
    // Use original filename with timestamp to avoid collisions
    const ext = path.extname(file.originalname);
    const name = path.basename(file.originalname, ext);
    cb(null, `${name}-${Date.now()}${ext}`);
  }
});

const upload = multer({
  storage,
  limits: { fileSize: 50 * 1024 * 1024 }, // 50MB limit
  fileFilter: (req, file, cb) => {
    // Accept images, PDFs, and videos
    const allowedTypes = /jpeg|jpg|png|gif|webp|pdf|mp4|mov|webm/;
    const ext = allowedTypes.test(path.extname(file.originalname).toLowerCase());
    const mime = allowedTypes.test(file.mimetype);
    if (ext && mime) {
      cb(null, true);
    } else {
      cb(new Error('Only images, PDFs, and videos are allowed'));
    }
  }
});

// Serve static files from public directory
app.use(express.static('public'));
app.use('/processed', express.static(PROCESSED_PATH));

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// Upload endpoint
app.post('/upload', upload.single('file'), (req, res) => {
  if (!req.file) {
    return res.status(400).json({ error: 'No file uploaded' });
  }

  const filename = req.file.filename;
  const ext = path.extname(filename);
  const basename = path.basename(filename, ext);

  res.json({
    message: 'File uploaded, processing...',
    filename,
    pending: path.join(UPLOAD_PATH, filename),
    processed: {
      original: path.join(PROCESSED_PATH, filename),
      thumbnail: path.join(PROCESSED_PATH, `${basename}.thumb.jpg`)
    },
    statusUrl: `/status/${filename}`
  });
});

// Upload multiple files
app.post('/upload/batch', upload.array('files', 10), (req, res) => {
  if (!req.files || req.files.length === 0) {
    return res.status(400).json({ error: 'No files uploaded' });
  }

  const files = req.files.map(file => {
    const ext = path.extname(file.filename);
    const basename = path.basename(file.filename, ext);
    return {
      filename: file.filename,
      pending: path.join(UPLOAD_PATH, file.filename),
      processed: path.join(PROCESSED_PATH, file.filename),
      thumbnail: path.join(PROCESSED_PATH, `${basename}.thumb.jpg`),
      statusUrl: `/status/${file.filename}`
    };
  });

  res.json({
    message: `${files.length} files uploaded, processing...`,
    files
  });
});

// Check file processing status
app.get('/status/:filename', (req, res) => {
  const filename = req.params.filename;
  const ext = path.extname(filename);
  const basename = path.basename(filename, ext);

  const pendingPath = path.join(UPLOAD_PATH, filename);
  const processingPath = path.join(UPLOAD_PATH.replace('pending', 'processing'), filename);
  const processedPath = path.join(PROCESSED_PATH, filename);
  const failedPath = path.join(FAILED_PATH, filename);
  const thumbnailPath = path.join(PROCESSED_PATH, `${basename}.thumb.jpg`);

  if (fs.existsSync(processedPath)) {
    return res.json({
      status: 'processed',
      file: processedPath,
      thumbnail: fs.existsSync(thumbnailPath) ? thumbnailPath : null,
      serveUrl: `/processed/${filename}`,
      thumbnailUrl: fs.existsSync(thumbnailPath) ? `/processed/${basename}.thumb.jpg` : null
    });
  }

  if (fs.existsSync(failedPath)) {
    const errorFile = `${failedPath}.error`;
    const error = fs.existsSync(errorFile) ? fs.readFileSync(errorFile, 'utf8') : 'Unknown error';
    return res.json({
      status: 'failed',
      error
    });
  }

  if (fs.existsSync(processingPath)) {
    return res.json({ status: 'processing' });
  }

  if (fs.existsSync(pendingPath)) {
    return res.json({ status: 'pending' });
  }

  return res.status(404).json({ status: 'not_found' });
});

// List all files
app.get('/files', (req, res) => {
  const pending = fs.existsSync(UPLOAD_PATH)
    ? fs.readdirSync(UPLOAD_PATH).filter(f => !f.startsWith('.'))
    : [];
  const processed = fs.existsSync(PROCESSED_PATH)
    ? fs.readdirSync(PROCESSED_PATH).filter(f => !f.startsWith('.'))
    : [];
  const failed = fs.existsSync(FAILED_PATH)
    ? fs.readdirSync(FAILED_PATH).filter(f => !f.startsWith('.') && !f.endsWith('.error'))
    : [];

  res.json({
    pending: pending.length,
    processed: processed.length,
    failed: failed.length,
    files: {
      pending,
      processed,
      failed
    }
  });
});

// Start server
app.listen(port, () => {
  console.log(`File processor example listening on port ${port}`);
  console.log(`Upload path: ${UPLOAD_PATH}`);
  console.log(`Processed path: ${PROCESSED_PATH}`);
  console.log(`Failed path: ${FAILED_PATH}`);
});
