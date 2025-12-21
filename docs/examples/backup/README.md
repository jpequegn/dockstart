# Database Backup Example

This example demonstrates dockstart's automatic database backup sidecar generation for a Node.js application using PostgreSQL.

## What This Shows

When you run `dockstart` on this project, it will:

1. Detect Node.js 20 from `package.json`
2. Detect PostgreSQL from the `pg` dependency
3. Generate a backup sidecar container with:
   - `Dockerfile.backup` - Alpine-based image with database clients
   - `backup.sh` - Main backup orchestration script
   - `backup-postgres.sh` - PostgreSQL-specific backup script
   - `restore-postgres.sh` - PostgreSQL restore script
   - `crontab` - Backup schedule (daily at 3 AM by default)

## Quick Start

```bash
# Generate devcontainer files
cd docs/examples/backup
dockstart .

# Or preview without writing
dockstart --dry-run .
```

## Expected Output

```
📂 Analyzing .
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres]
   💾 Backup: enabled (daily at 3 AM)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.backup...
📝 Generating backup scripts...

✨ Done!
```

## Generated Files

After running dockstart, you'll have:

```
.devcontainer/
├── devcontainer.json     # VS Code configuration
├── docker-compose.yml    # App + PostgreSQL + Backup services
├── Dockerfile            # Node.js development image
├── Dockerfile.backup     # Backup sidecar image
├── crontab               # Backup schedule
├── entrypoint.sh         # Backup container startup
├── scripts/
│   ├── backup.sh         # Main backup orchestrator
│   ├── backup-postgres.sh
│   └── restore-postgres.sh
└── backups/
    └── .gitkeep
```

## Using the Dev Container

1. Open the project in VS Code
2. Click "Reopen in Container" when prompted
3. The app, PostgreSQL, and backup sidecar will start automatically

## Testing the Backup Feature

### Step 1: Seed the Database

```bash
# Inside the dev container
npm run db:seed

# Verify data exists
curl http://localhost:3000/notes
```

### Step 2: Add Some Data

```bash
# Create a new note
curl -X POST http://localhost:3000/notes \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Note", "content": "This note will be backed up!"}'

# Verify it was created
curl http://localhost:3000/notes
```

### Step 3: Trigger a Manual Backup

```bash
# Run backup immediately (don't wait for scheduled time)
docker compose exec db-backup /usr/local/bin/backup.sh

# View backup files
ls -la .devcontainer/backups/
```

### Step 4: Simulate Data Loss

```bash
# Delete all notes (simulating accidental data loss)
curl -X DELETE http://localhost:3000/notes/1
curl -X DELETE http://localhost:3000/notes/2
# ... or truncate directly
docker compose exec postgres psql -U postgres -d backup-example_dev -c "TRUNCATE notes"

# Verify data is gone
curl http://localhost:3000/notes
```

### Step 5: Restore from Backup

```bash
# Find the backup file
BACKUP=$(ls -t .devcontainer/backups/postgres-*.sql.gz | head -1)
echo "Restoring from: $BACKUP"

# Stop the app to prevent writes
docker compose stop app

# Restore the backup
gunzip -c "$BACKUP" | docker compose exec -T postgres psql -U postgres -d backup-example_dev

# Restart the app
docker compose start app

# Verify data is restored!
curl http://localhost:3000/notes
```

## Backup Schedule

By default, backups run daily at 3 AM. You can customize this by editing `.devcontainer/crontab`:

```bash
# Every 6 hours
0 */6 * * * /usr/local/bin/backup.sh

# Every hour (for testing)
0 * * * * /usr/local/bin/backup.sh

# Every 5 minutes (for demo)
*/5 * * * * /usr/local/bin/backup.sh
```

After editing, rebuild the backup container:

```bash
docker compose build db-backup
docker compose up -d db-backup
```

## Viewing Backup Logs

```bash
# View backup container logs
docker compose logs db-backup

# Follow logs in real-time
docker compose logs -f db-backup
```

## Backup Retention

By default, backups older than 7 days are automatically deleted. Configure this via environment variable:

```yaml
# In docker-compose.yml
db-backup:
  environment:
    - RETENTION_DAYS=14  # Keep 2 weeks
```

## API Reference

This example includes a simple notes API:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/notes` | List all notes |
| GET | `/notes/:id` | Get single note |
| POST | `/notes` | Create note |
| PUT | `/notes/:id` | Update note |
| DELETE | `/notes/:id` | Delete note |

## Customization

After generation, you can modify:

- **Backup schedule**: Edit `.devcontainer/crontab`
- **Retention period**: Change `RETENTION_DAYS` environment variable
- **Backup scripts**: Customize `.devcontainer/scripts/backup-postgres.sh`

## Troubleshooting

### Backup container not starting

```bash
# Check container status
docker compose ps db-backup

# View logs
docker compose logs db-backup
```

### Database connection errors

```bash
# Test connectivity from backup container
docker compose exec db-backup pg_isready -h postgres -U postgres
```

### Backup files not appearing

```bash
# Check backup directory permissions
ls -la .devcontainer/backups/

# Run backup manually and check output
docker compose exec db-backup /usr/local/bin/backup.sh
```

## See Also

- [Database Backup Sidecar Documentation](../../sidecars/backup.md)
- [ADR-003: Database Backup Sidecar Architecture](../../adr/003-database-backup-sidecar.md)
