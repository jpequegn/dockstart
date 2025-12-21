# File Processor Example

This example demonstrates dockstart's automatic file processor sidecar generation for a Node.js image upload application.

## What This Shows

When you run `dockstart` on this project, it will:

1. Detect Node.js 20 from `package.json`
2. Detect multer from the dependencies (file upload library)
3. Generate a file processor sidecar container with:
   - `Dockerfile.processor` - Alpine-based image with ImageMagick, Poppler, FFmpeg
   - `process-files.sh` - Main file processing loop
   - `process-image.sh` - Image resize and thumbnail generation
   - `process-document.sh` - PDF text extraction
   - `process-video.sh` - Video thumbnail generation

## Quick Start

```bash
# Generate devcontainer files
cd docs/examples/file-processor
dockstart .

# Or preview without writing
dockstart --dry-run .
```

## Expected Output

```
📂 Analyzing .
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📁 Uploads: multer detected
   🔗 Sidecars: [file-processor]

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.processor...
📝 Generating processing scripts...

✨ Done!
```

## Generated Files

After running dockstart, you'll have:

```
.devcontainer/
├── devcontainer.json        # VS Code configuration
├── docker-compose.yml       # App + file-processor services
├── Dockerfile               # Node.js development image
├── Dockerfile.processor     # File processor sidecar image
├── entrypoint.processor.sh  # Processor startup script
├── scripts/
│   ├── process-files.sh     # Main processing loop
│   ├── process-image.sh     # Image processing
│   ├── process-document.sh  # PDF processing
│   └── process-video.sh     # Video processing
└── files/
    ├── pending/             # Input directory
    ├── processing/          # In-progress
    ├── processed/           # Output directory
    └── failed/              # Failed files
```

## Using the Dev Container

1. Open the project in VS Code
2. Click "Reopen in Container" when prompted
3. The app and file processor sidecar will start automatically

## Testing the File Processor

### Step 1: Start the Application

```bash
# Inside the dev container
npm install
npm start
```

The app runs on http://localhost:3000

### Step 2: Open the Web UI

Visit http://localhost:3000 in your browser. You'll see a drag-and-drop upload interface.

### Step 3: Upload an Image

Either:
- Drag an image file onto the upload area
- Click to browse and select a file

### Step 4: Watch the Processing

The log at the bottom shows:
1. File upload
2. Status polling
3. Processing completion

### Step 5: View Processed Files

The "Processed Files" section shows thumbnails of processed images.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| POST | `/upload` | Upload single file |
| POST | `/upload/batch` | Upload multiple files |
| GET | `/status/:filename` | Check processing status |
| GET | `/files` | List all files |
| GET | `/processed/:filename` | Serve processed file |

### Upload a File with curl

```bash
# Upload an image
curl -F "file=@photo.jpg" http://localhost:3000/upload

# Response:
{
  "message": "File uploaded, processing...",
  "filename": "photo-1703123456789.jpg",
  "pending": "/uploads/pending/photo-1703123456789.jpg",
  "processed": {
    "original": "/uploads/processed/photo-1703123456789.jpg",
    "thumbnail": "/uploads/processed/photo-1703123456789.thumb.jpg"
  },
  "statusUrl": "/status/photo-1703123456789.jpg"
}
```

### Check Processing Status

```bash
curl http://localhost:3000/status/photo-1703123456789.jpg

# While processing:
{ "status": "processing" }

# When done:
{
  "status": "processed",
  "file": "/uploads/processed/photo-1703123456789.jpg",
  "thumbnail": "/uploads/processed/photo-1703123456789.thumb.jpg",
  "serveUrl": "/processed/photo-1703123456789.jpg"
}
```

### List All Files

```bash
curl http://localhost:3000/files

{
  "pending": 0,
  "processed": 5,
  "failed": 0,
  "files": {
    "pending": [],
    "processed": ["photo.jpg", "photo.thumb.jpg", ...],
    "failed": []
  }
}
```

## Processing Pipeline

1. **App saves file** to `/uploads/pending/image.jpg`
2. **Processor detects** new file (polls every 5 seconds)
3. **Moves to processing** → `/uploads/processing/image.jpg`
4. **Processes file**:
   - Images: Resize to max 2000px, generate 200x200 thumbnail
   - PDFs: Extract text, create first-page thumbnail
   - Videos: Create thumbnail at 1s, generate 3s GIF preview
5. **Moves to processed** → `/uploads/processed/image.jpg`
6. **App receives** notification (optional webhook)

## Viewing Processor Logs

```bash
# View file processor logs
docker compose logs file-processor

# Follow logs in real-time
docker compose logs -f file-processor
```

## Common Operations

### Manually Trigger Processing

```bash
docker compose exec file-processor /scripts/process-files.sh
```

### Process a Specific File

```bash
docker compose exec file-processor /scripts/process-image.sh /uploads/pending/photo.jpg
```

### Reprocess Failed Files

```bash
# Move failed files back to pending
docker compose exec file-processor mv /uploads/failed/* /uploads/pending/
```

### Clear All Files

```bash
docker compose exec file-processor rm -rf /uploads/pending/* /uploads/processed/* /uploads/failed/*
```

## Customization

### Change Thumbnail Size

Edit `.devcontainer/scripts/process-image.sh`:

```bash
THUMBNAIL_SIZE="${THUMBNAIL_SIZE:-300x300}"  # Change from 200x200
```

### Adjust Poll Interval

Edit `.devcontainer/docker-compose.yml`:

```yaml
file-processor:
  environment:
    - POLL_INTERVAL=2  # Check every 2 seconds instead of 5
```

### Enable Video Processing

Video processing is enabled by default. The processor will:
- Generate thumbnail at 1 second
- Create 3-second GIF preview
- Extract metadata to JSON

### Use inotify (Linux)

For instant file detection on Linux:

```yaml
file-processor:
  environment:
    - USE_INOTIFY=true
```

## Troubleshooting

### Files Not Being Processed

```bash
# Check processor container is running
docker compose ps file-processor

# Check pending directory
docker compose exec file-processor ls -la /uploads/pending

# View logs
docker compose logs file-processor
```

### Processing Fails

```bash
# Check failed directory
docker compose exec file-processor ls -la /uploads/failed

# View error file
docker compose exec file-processor cat /uploads/failed/image.jpg.error
```

### Memory Issues

Large files may require more memory. Increase limits:

```yaml
file-processor:
  deploy:
    resources:
      limits:
        memory: 1G  # Default is 512M
```

## File Types Supported

| Type | Extensions | Processing |
|------|------------|------------|
| Images | jpg, png, gif, webp, tiff, bmp | Resize, thumbnail, optimize |
| Documents | pdf | Text extraction, thumbnail |
| Videos | mp4, mov, avi, webm, mkv | Thumbnail, GIF preview, metadata |

## See Also

- [File Processor Sidecar Documentation](../../sidecars/file-processor.md)
- [ADR-004: File Processing Sidecar Architecture](../../adr/004-file-processing-sidecar.md)
