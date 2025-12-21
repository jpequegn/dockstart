# ADR-004: File Processing Sidecar Architecture

**Status**: Accepted
**Date**: 2025-12-20
**Issue**: #43

## Context

Dockstart generates Docker dev environments for various project types. Many applications need to process uploaded files (image resizing, PDF extraction, video transcoding, virus scanning). Currently, developers must manually configure file processing infrastructure.

### Problem Statement

Developers need:
1. Automated file processing without manual infrastructure setup
2. Language-agnostic processing (works with Node.js, Go, Python, Rust)
3. Clear file flow with status tracking (pending → processing → processed → failed)
4. Graceful failure handling and retry mechanisms
5. Resource-efficient processing that doesn't block the main application

## Research: File Watching Approaches

### 1. inotify-based (inotifywait)

```bash
inotifywait -m -e create,moved_to /uploads/pending/ | while read path action file; do
    process_file "$path$file"
done
```

| Aspect | Details |
|--------|---------|
| **Mechanism** | Linux kernel inotify API |
| **Efficiency** | Event-driven, no polling overhead |
| **Latency** | Near-instant (<10ms) |
| **Container Support** | Works with bind mounts, limited with named volumes on macOS |

**Pros**:
- Zero CPU usage when idle
- Immediate file detection
- Native Linux support

**Cons**:
- Linux-only (not macOS native)
- Limited inotify watches (default 8192)
- May miss events during container restart
- Doesn't work reliably with Docker for Mac volume mounts

**Recommendation**: Best for Linux-only deployments.

### 2. Polling-based (find/stat loop)

```bash
while true; do
    find /uploads/pending -type f -mmin +1 -exec process_file {} \;
    sleep 5
done
```

| Aspect | Details |
|--------|---------|
| **Mechanism** | Periodic filesystem scan |
| **Efficiency** | CPU overhead proportional to file count |
| **Latency** | Poll interval (1-30 seconds typical) |
| **Container Support** | Universal (all platforms, all volume types) |

**Pros**:
- Works everywhere (Linux, macOS, Windows)
- Simple, reliable implementation
- Works with all Docker volume types
- Survives container restarts

**Cons**:
- CPU overhead (minimal for small directories)
- Latency based on poll interval
- More disk I/O than event-driven

**Recommendation**: Best for cross-platform dev environments (dockstart's target).

### 3. Message Queue Pattern (Redis pub/sub)

```yaml
# App publishes file path to Redis
redis-cli PUBLISH file:uploaded "/uploads/pending/image.jpg"

# Processor subscribes and processes
redis-cli SUBSCRIBE file:uploaded
```

| Aspect | Details |
|--------|---------|
| **Mechanism** | Application publishes events |
| **Efficiency** | No filesystem scanning |
| **Latency** | Near-instant |
| **Container Support** | Universal |

**Pros**:
- Decoupled architecture
- Reliable delivery with acknowledgments
- Easy to scale horizontally
- Works with distributed systems

**Cons**:
- Requires application modification
- Adds Redis dependency
- More complex setup
- Overkill for simple file processing

**Recommendation**: Best for production systems with existing Redis.

### 4. Hybrid Approach (Recommended for dockstart)

Combine polling with optional Redis integration:

```bash
# Default: Polling (works everywhere)
while true; do
    find /files/pending -type f -mmin +0 | while read file; do
        process_file "$file"
    done
    sleep "${POLL_INTERVAL:-5}"
done

# Optional: Subscribe to Redis for instant processing
if [ -n "$REDIS_URL" ]; then
    redis-cli SUBSCRIBE file:uploaded | while read type channel message; do
        [ "$type" = "message" ] && process_file "$message"
    done
fi
```

## Research: Shared Volume Patterns

### Volume Structure

```
/files/                     # Shared volume root
├── pending/                # App writes files here
│   └── upload-123.jpg
├── processing/             # Processor moves files here during work
│   └── upload-456.jpg
├── processed/              # Successfully processed files
│   ├── upload-789.jpg
│   └── upload-789.thumb.jpg
└── failed/                 # Failed processing (with error log)
    ├── upload-999.jpg
    └── upload-999.error
```

### Docker Compose Configuration

```yaml
services:
  app:
    volumes:
      - files:/files
    environment:
      - UPLOAD_PATH=/files/pending

  file-processor:
    volumes:
      - files:/files
    environment:
      - PENDING_PATH=/files/pending
      - PROCESSING_PATH=/files/processing
      - PROCESSED_PATH=/files/processed
      - FAILED_PATH=/files/failed

volumes:
  files:
```

### File Handling Best Practices

1. **Atomic moves**: Use `mv` not `cp` to prevent partial reads
2. **Unique filenames**: Include timestamp/UUID to prevent collisions
3. **Lock files**: Create `.lock` file during processing
4. **Completion markers**: Create `.done` file when finished

```bash
# Atomic processing pattern
LOCK="/files/processing/${filename}.lock"
touch "$LOCK"
mv "/files/pending/$filename" "/files/processing/$filename"
process_file "/files/processing/$filename"
mv "/files/processing/$filename" "/files/processed/$filename"
rm "$LOCK"
```

## Research: Processing Tools

### Image Processing

| Tool | Container Size | Use Case | Command |
|------|---------------|----------|---------|
| ImageMagick | ~50MB | Resize, convert, optimize | `convert input.jpg -resize 200x200 thumb.jpg` |
| jpegoptim | ~5MB | JPEG optimization | `jpegoptim --max=80 image.jpg` |
| pngquant | ~3MB | PNG optimization | `pngquant --quality=65-80 image.png` |
| libvips | ~30MB | Fast image processing | `vipsthumbnail input.jpg -s 200` |

**Recommendation**: ImageMagick for versatility, libvips for performance.

### Document Processing

| Tool | Container Size | Use Case | Command |
|------|---------------|----------|---------|
| Poppler | ~15MB | PDF text extraction | `pdftotext input.pdf output.txt` |
| pdftk | ~50MB | PDF manipulation | `pdftk input.pdf cat 1-5 output out.pdf` |
| LibreOffice | ~400MB | Document conversion | `libreoffice --headless --convert-to pdf doc.docx` |
| Pandoc | ~80MB | Format conversion | `pandoc input.md -o output.pdf` |

**Recommendation**: Poppler for PDF extraction, skip LibreOffice for dev (too heavy).

### Video/Audio Processing

| Tool | Container Size | Use Case | Command |
|------|---------------|----------|---------|
| FFmpeg | ~100MB | Transcoding, thumbnails | `ffmpeg -i input.mp4 -vf scale=320:-1 thumb.gif` |
| MediaInfo | ~5MB | Metadata extraction | `mediainfo --Output=JSON input.mp4` |

**Recommendation**: FFmpeg covers 95% of video/audio needs.

### Security Scanning

| Tool | Container Size | Use Case | Command |
|------|---------------|----------|---------|
| ClamAV | ~300MB + signatures | Virus scanning | `clamscan --infected --remove input.jpg` |
| YARA | ~10MB | Pattern matching | `yara rules.yar input.jpg` |

**Recommendation**: ClamAV for full scanning (optional, heavy).

## Decision

### Architecture: Modular File Processor Sidecar

Use a configurable processor container that handles multiple file types:

```yaml
services:
  file-processor:
    build:
      context: .devcontainer
      dockerfile: Dockerfile.processor
    volumes:
      - files:/files
    environment:
      - POLL_INTERVAL=5
      - PROCESS_IMAGES=true
      - PROCESS_DOCUMENTS=false
      - PROCESS_VIDEO=false
      - SCAN_VIRUSES=false
      - MAX_FILE_SIZE=50M
      - RETRY_COUNT=3
      - RETRY_DELAY=10
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    restart: unless-stopped
```

### Container Image: Alpine-Based with Modular Tools

Target size: 50-150MB depending on enabled features.

```dockerfile
FROM alpine:3.19

# Base tools (always included)
RUN apk add --no-cache \
    bash \
    coreutils \
    findutils \
    file

# Image processing (conditional)
ARG ENABLE_IMAGES=true
RUN if [ "$ENABLE_IMAGES" = "true" ]; then \
    apk add --no-cache imagemagick jpegoptim pngquant; \
    fi

# Document processing (conditional)
ARG ENABLE_DOCUMENTS=false
RUN if [ "$ENABLE_DOCUMENTS" = "true" ]; then \
    apk add --no-cache poppler-utils; \
    fi

# Video processing (conditional)
ARG ENABLE_VIDEO=false
RUN if [ "$ENABLE_VIDEO" = "true" ]; then \
    apk add --no-cache ffmpeg; \
    fi

COPY scripts/ /usr/local/bin/
COPY entrypoint.sh /entrypoint.sh

VOLUME ["/files"]
ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/local/bin/processor.sh"]
```

### Processing Pipeline

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│    App      │────▶│   /pending   │────▶│ file-processor  │
│ (uploads)   │     │              │     │   (watcher)     │
└─────────────┘     └──────────────┘     └────────┬────────┘
                                                   │
                    ┌──────────────────────────────┼──────────────────────────────┐
                    │                              │                              │
                    ▼                              ▼                              ▼
            ┌──────────────┐              ┌──────────────┐              ┌──────────────┐
            │ /processing  │              │ /processed   │              │   /failed    │
            │  (working)   │─────────────▶│  (success)   │              │   (error)    │
            └──────────────┘              └──────────────┘              └──────────────┘
                                                  │
                                                  ▼
                                          ┌──────────────┐
                                          │  Callback    │
                                          │ (webhook/    │
                                          │  file/queue) │
                                          └──────────────┘
```

### File Flow States

| State | Directory | Description |
|-------|-----------|-------------|
| **Pending** | `/files/pending/` | App uploads files here |
| **Processing** | `/files/processing/` | Processor working on file |
| **Processed** | `/files/processed/` | Successfully completed |
| **Failed** | `/files/failed/` | Processing failed after retries |

### Callback/Notification Mechanisms

Support three notification methods (configurable):

#### 1. File-based (Default, Simplest)

```bash
# Create .done file with metadata
cat > "/files/processed/${filename}.done" <<EOF
{
  "original": "upload-123.jpg",
  "processed": ["upload-123.thumb.jpg", "upload-123.optimized.jpg"],
  "timestamp": "2025-12-20T10:30:00Z",
  "duration_ms": 1250
}
EOF
```

App polls for `.done` files or uses inotify if available.

#### 2. Webhook (HTTP callback)

```bash
# POST to app when processing completes
curl -X POST "${WEBHOOK_URL}" \
  -H "Content-Type: application/json" \
  -d '{
    "event": "file.processed",
    "file": "upload-123.jpg",
    "outputs": ["upload-123.thumb.jpg"],
    "status": "success"
  }'
```

#### 3. Redis Pub/Sub (When Redis available)

```bash
# Publish completion event
redis-cli PUBLISH file:processed '{
  "file": "upload-123.jpg",
  "status": "success",
  "outputs": ["upload-123.thumb.jpg"]
}'
```

### Failure Handling

```bash
# processor.sh - Retry logic
process_with_retry() {
    local file="$1"
    local retries="${RETRY_COUNT:-3}"
    local delay="${RETRY_DELAY:-10}"

    for attempt in $(seq 1 $retries); do
        if process_file "$file"; then
            return 0
        fi
        echo "Attempt $attempt failed, retrying in ${delay}s..."
        sleep "$delay"
    done

    # Move to failed with error log
    mv "$file" "/files/failed/"
    echo "Processing failed after $retries attempts" > "/files/failed/${file##*/}.error"
    return 1
}
```

### Resource Limits

Default limits for dev environments:

```yaml
deploy:
  resources:
    limits:
      cpus: '1.0'      # 1 CPU core
      memory: 512M     # 512MB RAM
    reservations:
      cpus: '0.25'     # Minimum 0.25 cores
      memory: 128M     # Minimum 128MB
```

Processing-specific limits:

| Operation | Recommended Memory | CPU |
|-----------|-------------------|-----|
| Image resize | 256MB | 0.5 |
| Image optimize | 128MB | 0.25 |
| PDF extraction | 256MB | 0.5 |
| Video transcode | 1GB+ | 1.0+ |
| Virus scan | 512MB+ | 0.5 |

## Detection Rules

### When to Add File Processor

| Detection | Condition | Default Operations |
|-----------|-----------|-------------------|
| Image upload library | multer, formidable, gin-contrib/static | Image resize, optimize |
| File storage config | S3_BUCKET, UPLOAD_PATH env | General file processing |
| Image manipulation lib | sharp, pillow, imaging | Image processing |
| PDF library | pdf-parse, pypdf, pdfcpu | Document extraction |

### ProcessorSidecarConfig

```go
type ProcessorSidecarConfig struct {
    // Enable specific processing types
    ProcessImages    bool
    ProcessDocuments bool
    ProcessVideo     bool
    ScanViruses      bool

    // Polling configuration
    PollInterval int // seconds (default: 5)

    // File paths
    PendingPath    string // default: /files/pending
    ProcessingPath string // default: /files/processing
    ProcessedPath  string // default: /files/processed
    FailedPath     string // default: /files/failed

    // Processing options
    MaxFileSize     string // default: 50M
    RetryCount      int    // default: 3
    RetryDelay      int    // seconds (default: 10)

    // Notification method
    NotifyMethod string // file, webhook, redis
    WebhookURL   string // for webhook method

    // Resource limits
    CPULimit    string // default: 1.0
    MemoryLimit string // default: 512M

    // Image-specific options
    ThumbnailSize  string // default: 200x200
    OptimizeJPEG   bool   // default: true
    OptimizePNG    bool   // default: true
    JPEGQuality    int    // default: 80
}
```

## Generated Files

### Dockerfile.processor

```dockerfile
FROM alpine:3.19

RUN apk add --no-cache \
    bash \
    coreutils \
    findutils \
    file \
    imagemagick \
    jpegoptim \
    pngquant

COPY scripts/processor.sh /usr/local/bin/
COPY scripts/process-image.sh /usr/local/bin/
COPY entrypoint.processor.sh /entrypoint.sh

RUN chmod +x /usr/local/bin/*.sh /entrypoint.sh

VOLUME ["/files"]
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD pgrep -f processor.sh || exit 1

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/usr/local/bin/processor.sh"]
```

### processor.sh (Main Loop)

```bash
#!/bin/bash
set -eo pipefail

PENDING="${PENDING_PATH:-/files/pending}"
PROCESSING="${PROCESSING_PATH:-/files/processing}"
PROCESSED="${PROCESSED_PATH:-/files/processed}"
FAILED="${FAILED_PATH:-/files/failed}"
POLL_INTERVAL="${POLL_INTERVAL:-5}"

# Ensure directories exist
mkdir -p "$PENDING" "$PROCESSING" "$PROCESSED" "$FAILED"

echo "[$(date)] File processor started"
echo "[$(date)] Watching: $PENDING (poll: ${POLL_INTERVAL}s)"

while true; do
    # Find files older than 1 second (ensure upload complete)
    find "$PENDING" -type f -mmin +0.016 2>/dev/null | while read -r file; do
        filename=$(basename "$file")
        echo "[$(date)] Processing: $filename"

        # Move to processing
        mv "$file" "$PROCESSING/$filename"

        # Process based on file type
        if process_file "$PROCESSING/$filename"; then
            mv "$PROCESSING/$filename" "$PROCESSED/$filename"
            create_notification "$filename" "success"
            echo "[$(date)] Completed: $filename"
        else
            mv "$PROCESSING/$filename" "$FAILED/$filename"
            create_notification "$filename" "failed"
            echo "[$(date)] Failed: $filename"
        fi
    done

    sleep "$POLL_INTERVAL"
done
```

### process-image.sh

```bash
#!/bin/bash
# Process image files: resize and optimize

INPUT="$1"
OUTPUT_DIR="${PROCESSED_PATH:-/files/processed}"
THUMB_SIZE="${THUMBNAIL_SIZE:-200x200}"
JPEG_QUALITY="${JPEG_QUALITY:-80}"

filename=$(basename "$INPUT")
extension="${filename##*.}"
basename="${filename%.*}"

case "$extension" in
    jpg|jpeg|JPG|JPEG)
        # Create thumbnail
        convert "$INPUT" -resize "$THUMB_SIZE^" -gravity center -extent "$THUMB_SIZE" \
            "$OUTPUT_DIR/${basename}.thumb.jpg"

        # Optimize original
        jpegoptim --max="$JPEG_QUALITY" --strip-all "$INPUT"
        ;;

    png|PNG)
        # Create thumbnail
        convert "$INPUT" -resize "$THUMB_SIZE^" -gravity center -extent "$THUMB_SIZE" \
            "$OUTPUT_DIR/${basename}.thumb.png"

        # Optimize original
        pngquant --quality=65-80 --force --output "$INPUT" "$INPUT"
        ;;

    gif|GIF)
        # Create static thumbnail from first frame
        convert "$INPUT[0]" -resize "$THUMB_SIZE^" -gravity center -extent "$THUMB_SIZE" \
            "$OUTPUT_DIR/${basename}.thumb.jpg"
        ;;

    *)
        echo "Unsupported image format: $extension"
        return 1
        ;;
esac

echo "Processed: $filename"
```

## Implementation Plan

### Phase 1: Core Infrastructure (Issue #44)
- [ ] Create `ProcessorSidecarConfig` in models
- [ ] Implement file processor detection rules
- [ ] Create processor sidecar Dockerfile template

### Phase 2: Processing Scripts (Issue #45)
- [ ] Generate image processing scripts
- [ ] Implement polling-based file watcher
- [ ] Create notification mechanism (file-based default)

### Phase 3: Integration (Issue #46)
- [ ] Update compose generator for processor sidecar
- [ ] Add processor configuration options
- [ ] Create documentation and examples

### Phase 4: Extended Features (Future)
- [ ] Document processing (PDF extraction)
- [ ] Video processing (thumbnail generation)
- [ ] Virus scanning (ClamAV integration)
- [ ] Redis pub/sub notifications

## Consequences

### Positive
- Automated file processing without developer configuration
- Language-agnostic (works with any app framework)
- Clear file flow with status tracking
- Graceful failure handling with retries
- Resource limits prevent runaway processing
- Cross-platform compatibility (polling-based)

### Negative
- Additional container resource usage
- Processing latency (poll interval)
- Disk space for processed files
- More complex docker-compose

### Neutral
- Polling vs event-driven trade-off (reliability over latency)
- File-based notifications by default (simplest)
- Image processing enabled by default, others opt-in

## Alternatives Considered

### 1. Application-Level Processing

Process files within the application code.

**Rejected**: Language-specific, blocks main application, harder to resource-limit.

### 2. External Processing Service

Use cloud services (AWS Lambda, Cloudinary).

**Rejected**: Requires external dependencies, not suitable for local dev.

### 3. inotify-Only Approach

Use inotify exclusively for file watching.

**Rejected**: Not cross-platform, unreliable with Docker for Mac volumes.

### 4. Message Queue Required

Require Redis/RabbitMQ for all file events.

**Rejected**: Adds complexity, overkill for simple dev workflows.

## References

- [inotify-tools Documentation](https://github.com/inotify-tools/inotify-tools)
- [ImageMagick Command-Line Processing](https://imagemagick.org/script/command-line-processing.php)
- [Docker Resource Constraints](https://docs.docker.com/config/containers/resource_constraints/)
- [FFmpeg Documentation](https://ffmpeg.org/documentation.html)
- [Alpine Package Repository](https://pkgs.alpinelinux.org/packages)
- [Container File Sharing Best Practices](https://docs.docker.com/storage/volumes/)
