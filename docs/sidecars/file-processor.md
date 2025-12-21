# File Processing Sidecar

Automatic file processing for uploaded files in your development environment.

## Overview

When dockstart detects file upload libraries in your project, it automatically generates a file processor sidecar that:

- Watches for new files in the `/uploads/pending` directory
- Processes files based on type (images, documents, videos)
- Outputs processed files to `/uploads/processed`
- Moves failed files to `/uploads/failed` with error details
- Runs in a resource-limited container (512MB RAM, 0.5 CPU by default)

## Quick Start

```bash
$ dockstart ./my-image-gallery

📂 Analyzing ./my-image-gallery...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]
   📁 Uploads: multer detected
   🔗 Sidecars: [file-processor]

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.processor...
📝 Generating processing scripts...

✨ Done!
```

## Detected Libraries

| Language | Libraries | Auto-Detection |
|----------|-----------|----------------|
| Node.js | multer, formidable, busboy, express-fileupload | package.json dependencies |
| Python | python-multipart, aiofiles, flask-uploads | pyproject.toml / requirements.txt |
| Go | multipart (standard library) | imports in .go files |
| Rust | actix-multipart, multer | Cargo.toml dependencies |

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│  App Container  │     │   File Volume   │     │  file-processor │
│                 │     │                 │     │                 │
│   multer /      │────▶│   /uploads/     │◀────│   ImageMagick   │
│   formidable    │     │   pending/      │     │   Poppler       │
│                 │     │                 │     │   FFmpeg        │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        │                       ▼                       │
        │              /uploads/processed               │
        │                       │                       │
        └───────────────────────┴───────────────────────┘
                    Shared Docker volume
```

### Processing Pipeline

1. **App writes file** → `/uploads/pending/image.jpg`
2. **Processor detects file** → Polls every 5 seconds (or inotify on Linux)
3. **Move to processing** → `/uploads/processing/image.jpg`
4. **Process file** → Apply transformations based on file type
5. **Move to processed** → `/uploads/processed/image.jpg` + `/uploads/processed/image.thumb.jpg`
6. **Notify app** → Touch notification file or HTTP webhook

## Generated Files

```
.devcontainer/
├── docker-compose.yml       # Includes file-processor service
├── Dockerfile.processor     # Alpine + ImageMagick + Poppler + FFmpeg
├── entrypoint.processor.sh  # Container startup
├── scripts/
│   ├── process-files.sh     # Main processing loop
│   ├── process-image.sh     # Image processing (resize, thumbnail)
│   ├── process-document.sh  # PDF text extraction
│   └── process-video.sh     # Video thumbnails, previews
└── files/
    ├── pending/             # Input directory
    ├── processing/          # In-progress
    ├── processed/           # Output directory
    └── failed/              # Failed files
```

## Generated Docker Compose Service

```yaml
services:
  app:
    volumes:
      - uploads:/uploads
    environment:
      - UPLOAD_PATH=/uploads/pending
      - PROCESSED_PATH=/uploads/processed
      - FAILED_PATH=/uploads/failed

  file-processor:
    build:
      context: .
      dockerfile: Dockerfile.processor
    volumes:
      - uploads:/uploads
    depends_on:
      - app
    environment:
      - PENDING_PATH=/uploads/pending
      - PROCESSING_PATH=/uploads/processing
      - PROCESSED_PATH=/uploads/processed
      - FAILED_PATH=/uploads/failed
      - POLL_INTERVAL=5
      - MAX_FILE_SIZE=52428800
      - RETRY_COUNT=3
      - NOTIFY_METHOD=file
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
    restart: unless-stopped

volumes:
  uploads:
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PENDING_PATH` | `/uploads/pending` | Directory for incoming files |
| `PROCESSING_PATH` | `/uploads/processing` | Temporary processing directory |
| `PROCESSED_PATH` | `/uploads/processed` | Output directory for processed files |
| `FAILED_PATH` | `/uploads/failed` | Directory for failed files |
| `POLL_INTERVAL` | `5` | Seconds between directory scans |
| `MAX_FILE_SIZE` | `52428800` | Maximum file size in bytes (50MB) |
| `RETRY_COUNT` | `3` | Number of retry attempts for failed files |
| `THUMBNAIL_SIZE` | `200x200` | Thumbnail dimensions |
| `NOTIFY_METHOD` | `file` | Notification method (`file` or `http`) |
| `NOTIFY_URL` | - | HTTP webhook URL (when NOTIFY_METHOD=http) |

### Processing Options

The processor supports three processing modes, configured at generation time:

| Option | Tools | Enabled by Default |
|--------|-------|-------------------|
| **Images** | ImageMagick, jpegoptim, pngquant | Yes |
| **Documents** | Poppler (pdftotext, pdftoppm) | No |
| **Video** | FFmpeg | No |

## Processing Details

### Image Processing

Supported formats: JPEG, PNG, GIF, WebP, TIFF, BMP

| Operation | Description | Output |
|-----------|-------------|--------|
| Copy original | Preserve original file | `{filename}.{ext}` |
| Resize | Resize to max 2000px width | `{filename}.{ext}` (replaced) |
| Thumbnail | Generate 200x200 thumbnail | `{filename}.thumb.jpg` |
| Optimize | Compress JPEGs (jpegoptim) / PNGs (pngquant) | In-place |

**Example output:**
```
Input:  /uploads/pending/photo.jpg (5MB, 4000x3000)
Output: /uploads/processed/photo.jpg (200KB, 2000x1500)
        /uploads/processed/photo.thumb.jpg (15KB, 200x200)
```

### Document Processing

Supported formats: PDF

| Operation | Description | Output |
|-----------|-------------|--------|
| Copy original | Preserve PDF | `{filename}.pdf` |
| Extract text | Full text extraction | `{filename}.txt` |
| Metadata | PDF info extraction | `{filename}.info` |
| Thumbnail | First page as image | `{filename}.thumb.jpg` |

**Example output:**
```
Input:  /uploads/pending/document.pdf
Output: /uploads/processed/document.pdf
        /uploads/processed/document.txt
        /uploads/processed/document.info
        /uploads/processed/document.thumb.jpg
```

### Video Processing

Supported formats: MP4, MOV, AVI, WebM, MKV

| Operation | Description | Output |
|-----------|-------------|--------|
| Copy original | Preserve video | `{filename}.{ext}` |
| Thumbnail | Capture at 1 second | `{filename}.thumb.jpg` |
| Preview GIF | First 3 seconds | `{filename}.preview.gif` |
| Metadata | Full metadata JSON | `{filename}.info.json` |

**Example output:**
```
Input:  /uploads/pending/video.mp4
Output: /uploads/processed/video.mp4
        /uploads/processed/video.thumb.jpg
        /uploads/processed/video.preview.gif
        /uploads/processed/video.info.json
```

## Common Operations

### View Processor Logs

```bash
# View logs
docker compose logs file-processor

# Follow logs in real-time
docker compose logs -f file-processor

# View only errors
docker compose logs file-processor 2>&1 | grep ERROR
```

### Check Processing Status

```bash
# Count pending files
ls /uploads/pending | wc -l

# Count processed files
ls /uploads/processed | wc -l

# Count failed files
ls /uploads/failed | wc -l
```

### Manually Trigger Processing

```bash
# Process a specific file
docker compose exec file-processor /scripts/process-image.sh /uploads/pending/myfile.jpg
```

### Reprocess Failed Files

```bash
# Move failed files back to pending
docker compose exec file-processor mv /uploads/failed/* /uploads/pending/

# Or just one file
docker compose exec file-processor mv /uploads/failed/image.jpg /uploads/pending/
```

## Integration Examples

### Express.js with Multer

```javascript
const express = require('express');
const multer = require('multer');
const path = require('path');
const fs = require('fs');

const app = express();

// Configure multer to save to pending directory
const upload = multer({
  dest: process.env.UPLOAD_PATH || '/uploads/pending',
  limits: { fileSize: 50 * 1024 * 1024 } // 50MB
});

// Upload endpoint
app.post('/upload', upload.single('image'), (req, res) => {
  // Rename file to preserve extension
  const ext = path.extname(req.file.originalname);
  const newPath = req.file.path + ext;
  fs.renameSync(req.file.path, newPath);

  // Return expected processed path
  const processedPath = newPath
    .replace('pending', 'processed');

  res.json({
    message: 'File uploaded, processing...',
    pending: newPath,
    processed: processedPath,
    thumbnail: processedPath.replace(ext, '.thumb.jpg')
  });
});

// Check if file is processed
app.get('/status/:filename', (req, res) => {
  const processedPath = path.join(
    process.env.PROCESSED_PATH || '/uploads/processed',
    req.params.filename
  );

  if (fs.existsSync(processedPath)) {
    res.json({ status: 'ready', path: processedPath });
  } else {
    res.json({ status: 'processing' });
  }
});
```

### Python with FastAPI

```python
from fastapi import FastAPI, UploadFile
import shutil
import os

app = FastAPI()

UPLOAD_PATH = os.getenv('UPLOAD_PATH', '/uploads/pending')
PROCESSED_PATH = os.getenv('PROCESSED_PATH', '/uploads/processed')

@app.post("/upload")
async def upload_file(file: UploadFile):
    # Save to pending directory
    file_path = os.path.join(UPLOAD_PATH, file.filename)
    with open(file_path, "wb") as buffer:
        shutil.copyfileobj(file.file, buffer)

    return {
        "filename": file.filename,
        "pending": file_path,
        "processed": os.path.join(PROCESSED_PATH, file.filename)
    }

@app.get("/status/{filename}")
async def check_status(filename: str):
    processed_path = os.path.join(PROCESSED_PATH, filename)
    if os.path.exists(processed_path):
        return {"status": "ready", "path": processed_path}
    return {"status": "processing"}
```

### Go with Standard Library

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {
    file, header, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    uploadPath := os.Getenv("UPLOAD_PATH")
    if uploadPath == "" {
        uploadPath = "/uploads/pending"
    }

    dst, err := os.Create(filepath.Join(uploadPath, header.Filename))
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer dst.Close()

    io.Copy(dst, file)

    fmt.Fprintf(w, "Uploaded: %s\n", header.Filename)
}
```

## Custom Processing

### Adding a Custom Processor

Create a custom script in `.devcontainer/scripts/`:

```bash
#!/bin/bash
# .devcontainer/scripts/process-custom.sh
# Custom processor for .xyz files

INPUT="$1"
FILENAME=$(basename "$INPUT")
PROCESSED_DIR="${PROCESSED_PATH:-/uploads/processed}"

# Your custom processing logic here
cp "$INPUT" "$PROCESSED_DIR/$FILENAME"

echo "Processed: $FILENAME"
```

Then modify `process-files.sh` to call your processor:

```bash
case "$EXTENSION_LOWER" in
    xyz)
        /scripts/process-custom.sh "$FILE"
        ;;
    # ... other cases
esac
```

### Custom Image Sizes

Modify `.devcontainer/scripts/process-image.sh`:

```bash
# Change thumbnail size
THUMBNAIL_SIZE="${THUMBNAIL_SIZE:-300x300}"

# Add medium size variant
convert "$INPUT" -resize "800x800>" "$PROCESSED_DIR/${BASENAME}.medium.${EXTENSION}"
```

### Webhook Notifications

Configure HTTP notifications instead of file-based:

```yaml
# In docker-compose.yml
file-processor:
  environment:
    - NOTIFY_METHOD=http
    - NOTIFY_URL=http://app:3000/api/file-processed
```

Your app receives POST requests:

```json
{
  "file": "image.jpg",
  "original": "/uploads/processed/image.jpg",
  "thumbnail": "/uploads/processed/image.thumb.jpg",
  "status": "success"
}
```

## Troubleshooting

### Processor Not Running

```bash
# Check container status
docker compose ps file-processor

# View container logs
docker compose logs file-processor

# Check if entrypoint is running
docker compose exec file-processor ps aux
```

### Files Not Being Processed

```bash
# Check pending directory
docker compose exec file-processor ls -la /uploads/pending

# Check file permissions
docker compose exec file-processor stat /uploads/pending/*

# Run processing manually
docker compose exec file-processor /scripts/process-files.sh
```

### Processing Errors

```bash
# Check failed directory
docker compose exec file-processor ls -la /uploads/failed

# View error for specific file
docker compose exec file-processor cat /uploads/failed/image.jpg.error

# Check ImageMagick installation
docker compose exec file-processor convert --version

# Check FFmpeg installation
docker compose exec file-processor ffmpeg -version
```

### Memory Issues

```bash
# Check container memory usage
docker stats file-processor

# Increase memory limit in docker-compose.yml
file-processor:
  deploy:
    resources:
      limits:
        memory: 1G  # Increase from 512M
```

### Permission Issues

```bash
# Check volume permissions
docker compose exec file-processor ls -la /uploads

# Fix permissions (if needed)
docker compose exec file-processor chmod -R 755 /uploads
```

## Advanced Configuration

### Disable File Processor

Use Docker Compose profiles:

```yaml
file-processor:
  profiles:
    - processor  # Only starts with --profile processor
```

Then start without processor:
```bash
docker compose up -d
```

Or with processor:
```bash
docker compose --profile processor up -d
```

### Linux with inotify

For better performance on Linux, enable inotify-based watching:

```yaml
file-processor:
  environment:
    - USE_INOTIFY=true
```

This replaces polling with instant file detection.

### External Processing Service

For production, you might want to use external services:

```yaml
file-processor:
  environment:
    - CLOUDINARY_URL=cloudinary://...
    - USE_EXTERNAL=true
```

## Resource Considerations

| Processing Type | Memory Usage | CPU Usage |
|-----------------|--------------|-----------|
| Images (small) | ~50MB | Low |
| Images (large) | ~200MB | Medium |
| PDFs | ~100MB | Low |
| Videos | ~300MB | High |

Default limits (512MB, 0.5 CPU) are suitable for typical development use. Increase for larger files or batch processing.

## See Also

- [ADR-004: File Processing Sidecar Architecture](../adr/004-file-processing-sidecar.md)
- [Log Aggregator Sidecar](./log-aggregator.md)
- [Background Worker Sidecar](./background-worker.md)
- [Database Backup Sidecar](./backup.md)
