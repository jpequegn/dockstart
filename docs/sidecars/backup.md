# Database Backup Sidecar

Automatic database backups for your development environment.

## Overview

When dockstart detects a database service (PostgreSQL, MySQL, or Redis), it automatically generates a backup sidecar that:

- Runs automated backups on a configurable schedule (default: daily at 3 AM)
- Rotates old backups automatically (default: 7-day retention)
- Stores backups in the `./backups/` directory (mounted from host)
- Uses lightweight Alpine-based container with Supercronic scheduler
- Supports PostgreSQL, MySQL, Redis (SQLite planned)

## Quick Start

```bash
$ dockstart ./my-rails-app

📂 Analyzing ./my-rails-app...
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: [postgres, redis]
   💾 Backup: enabled (daily at 3 AM)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating Dockerfile.backup...
📝 Generating backup scripts...

✨ Done!
```

## Supported Databases

| Database | Backup Tool | Hot Backup | Container Overhead |
|----------|-------------|------------|-------------------|
| PostgreSQL | pg_dump | Yes | ~8MB |
| MySQL/MariaDB | mysqldump | Yes | ~12MB |
| Redis | redis-cli BGSAVE + docker cp | Yes | ~2MB |
| SQLite | VACUUM INTO | Planned | <1MB |

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│     App     │     │  Database   │     │  db-backup  │
│             │────▶│  (postgres, │◀────│  (alpine +  │
│             │     │   mysql,    │     │ supercronic)│
└─────────────┘     │   redis)    │     └──────┬──────┘
                    └─────────────┘            │
                                               ▼
                                    ┌─────────────────┐
                                    │ .devcontainer/  │
                                    │   backups/      │
                                    │ (host mount)    │
                                    └─────────────────┘
```

## Generated Files

```
.devcontainer/
├── docker-compose.yml     # Includes db-backup service
├── Dockerfile.backup      # Alpine + database clients + Supercronic
├── crontab                # Backup schedule
├── entrypoint.sh          # Container startup with health checks
├── scripts/
│   ├── backup.sh          # Main backup orchestrator
│   ├── backup-postgres.sh # PostgreSQL backup script
│   ├── backup-mysql.sh    # MySQL backup script
│   ├── backup-redis.sh    # Redis backup script
│   ├── restore-postgres.sh # PostgreSQL restore script
│   ├── restore-mysql.sh   # MySQL restore script
│   └── restore-redis.sh   # Redis restore script
└── backups/
    └── .gitkeep           # Ensures directory exists
```

## Generated Docker Compose Service

```yaml
services:
  # ... your app and database services ...

  db-backup:
    build:
      context: .
      dockerfile: Dockerfile.backup
    volumes:
      - ./backups:/backup
      - /var/run/docker.sock:/var/run/docker.sock:ro  # For Redis only
    depends_on:
      - postgres  # Or mysql, redis
    environment:
      - BACKUP_DIR=/backup
      - RETENTION_DAYS=7
      - DB_HOST=postgres
      - DB_USER=postgres
      - DB_PASSWORD=postgres
      - DB_NAME=myapp_dev
    restart: unless-stopped

volumes:
  backups:
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_DIR` | `/backup` | Directory for backup files |
| `RETENTION_DAYS` | `7` | Days to keep backups |
| `DB_HOST` | (detected) | Database hostname |
| `DB_USER` | (detected) | Database user |
| `DB_PASSWORD` | (detected) | Database password |
| `DB_NAME` | (detected) | Database name |
| `REDIS_HOST` | `redis` | Redis hostname |
| `REDIS_PORT` | `6379` | Redis port |

### Custom Schedule

The backup schedule is defined in `.devcontainer/crontab`. Default is daily at 3 AM:

```
0 3 * * * /usr/local/bin/backup.sh
```

**Schedule Examples:**

```bash
# Every 6 hours
0 */6 * * *

# Every night at midnight
0 0 * * *

# Every Sunday at 2 AM
0 2 * * 0

# Every 30 minutes (for testing)
*/30 * * * *
```

## Backup Files

Backups are stored in `.devcontainer/backups/` with the naming convention:

```
{database}-{YYYY-MM-DD}T{HH-MM-SS}.{ext}

Examples:
postgres-2025-12-20T03-00-00.sql.gz
mysql-2025-12-20T03-00-00.sql.gz
redis-2025-12-20T03-00-00.rdb
```

## Common Operations

### View Backups

```bash
# List backup files
ls -la .devcontainer/backups/

# From inside container
docker compose exec db-backup ls -la /backup
```

### Trigger Manual Backup

```bash
docker compose exec db-backup /usr/local/bin/backup.sh
```

### View Backup Logs

```bash
docker compose logs db-backup
docker compose logs -f db-backup  # Follow logs
```

### Check Backup Status

```bash
# View recent backup output
docker compose exec db-backup cat /var/log/backup.log

# Check if scheduler is running
docker compose exec db-backup ps aux | grep supercronic
```

## Restore Procedures

### PostgreSQL Restore

```bash
# Stop the app to prevent writes during restore
docker compose stop app

# Restore from backup (choose the backup file you want)
BACKUP_FILE=".devcontainer/backups/postgres-2025-12-20T03-00-00.sql.gz"

# Restore using psql
gunzip -c "$BACKUP_FILE" | \
  docker compose exec -T postgres psql -U postgres -d myapp_dev

# Or use the restore script
docker compose exec db-backup /scripts/restore-postgres.sh /backup/postgres-2025-12-20T03-00-00.sql.gz

# Restart the app
docker compose start app
```

### MySQL Restore

```bash
# Stop the app
docker compose stop app

# Restore from backup
BACKUP_FILE=".devcontainer/backups/mysql-2025-12-20T03-00-00.sql.gz"
gunzip -c "$BACKUP_FILE" | \
  docker compose exec -T mysql mysql -u root -pmysql myapp_dev

# Or use the restore script
docker compose exec db-backup /scripts/restore-mysql.sh /backup/mysql-2025-12-20T03-00-00.sql.gz

# Restart the app
docker compose start app
```

### Redis Restore

```bash
# Stop Redis to replace data file
docker compose stop redis

# Find the Redis data volume
docker volume inspect myapp_redis-data

# Restore (Redis needs the data directory)
cp .devcontainer/backups/redis-2025-12-20T03-00-00.rdb \
   /path/to/redis-volume/dump.rdb

# Or use the restore script (container must be running)
docker compose exec db-backup /scripts/restore-redis.sh /backup/redis-2025-12-20T03-00-00.rdb

# Restart Redis
docker compose start redis
```

## Troubleshooting

### Backup Not Running

```bash
# Check if container is running
docker compose ps db-backup

# Check Supercronic scheduler
docker compose exec db-backup ps aux | grep supercronic

# View logs for errors
docker compose logs db-backup
```

### Database Connection Errors

```bash
# Test database connectivity
docker compose exec db-backup pg_isready -h postgres -U postgres

# For MySQL
docker compose exec db-backup mysqladmin ping -h mysql -u root -pmysql

# For Redis
docker compose exec db-backup redis-cli -h redis ping
```

### Disk Space Issues

```bash
# Check backup directory size
du -sh .devcontainer/backups/

# Manually clean old backups
find .devcontainer/backups/ -name "*.gz" -mtime +7 -delete
```

### Permission Issues

```bash
# Ensure backup directory is writable
chmod 755 .devcontainer/backups/

# Check container user
docker compose exec db-backup whoami
```

## Advanced Configuration

### Disable Backup Sidecar

Remove from docker-compose.yml or use Docker Compose profiles:

```yaml
db-backup:
  profiles:
    - backup  # Only starts with --profile backup
```

Then start without backup:
```bash
docker compose up -d
```

Or with backup:
```bash
docker compose --profile backup up -d
```

### Custom Retention Period

Edit `.devcontainer/docker-compose.yml`:

```yaml
db-backup:
  environment:
    - RETENTION_DAYS=14  # Keep 2 weeks of backups
```

### External Backup Storage

For cloud storage, you can modify the backup script or add a sync step:

```bash
# Example: Sync to S3 after backup
docker compose exec db-backup sh -c \
  "aws s3 sync /backup s3://my-bucket/backups/"
```

## Implementation Details

### Why Supercronic?

We use [Supercronic](https://github.com/aptible/supercronic) instead of traditional cron because:

- **No syslog dependency**: Works in minimal containers
- **Proper signal handling**: Graceful shutdown with Docker
- **Stdout logging**: Container-friendly output
- **Simple configuration**: No daemon, runs in foreground

### Why Docker Socket for Redis?

Redis backups use `docker cp` to copy the RDB file from the Redis container because:

- Redis RDB files are binary and must be copied atomically
- BGSAVE creates a point-in-time snapshot
- Docker socket allows cross-container file operations

The socket is mounted read-only (`/var/run/docker.sock:ro`) for security.

### Health Checks

The entrypoint script waits for databases to be ready before starting backups:

```bash
# PostgreSQL
pg_isready -h postgres -U postgres

# MySQL
mysqladmin ping -h mysql -u root -pmysql

# Redis
redis-cli -h redis ping
```

## See Also

- [ADR-003: Database Backup Sidecar Architecture](../adr/003-database-backup-sidecar.md)
- [Log Aggregator Sidecar](./log-aggregator.md)
- [Background Worker Sidecar](./background-worker.md)
