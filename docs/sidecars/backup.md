# Database Backup Sidecar

Quick reference for the auto-generated database backup sidecar.

## Overview

When dockstart detects a database service, it can generate a backup sidecar that:
- Runs automated backups on a schedule (default: daily at 3 AM)
- Rotates old backups (default: 7-day retention)
- Stores backups in `./backups/` directory
- Supports PostgreSQL, MySQL, Redis, and SQLite

## Supported Databases

| Database | Backup Method | Hot Backup | Container Size |
|----------|---------------|------------|----------------|
| PostgreSQL | pg_dump | Yes | ~8MB |
| MySQL/MariaDB | mysqldump | Yes | ~12MB |
| Redis | RDB snapshot | Yes | ~2MB |
| SQLite | VACUUM INTO | No (requires stop) | <1MB |

## Generated Files

```
.devcontainer/
├── docker-compose.yml     # Includes db-backup service
├── Dockerfile.backup      # Alpine + database clients
└── scripts/
    ├── backup.sh          # Main backup script
    ├── rotate.sh          # Cleanup old backups
    └── restore.sh         # Restore helper
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKUP_SCHEDULE` | `0 3 * * *` | Cron schedule (3 AM daily) |
| `BACKUP_RETENTION_DAYS` | `7` | Days to keep backups |
| `DB_TYPE` | (detected) | postgres, mysql, redis, sqlite |
| `DB_CONTAINER` | (detected) | Database container name |
| `DB_NAME` | (detected) | Database name |
| `DB_USER` | (detected) | Database user |
| `DB_PASSWORD` | (detected) | Database password |

### Custom Schedule Examples

```yaml
# Every 6 hours
BACKUP_SCHEDULE: "0 */6 * * *"

# Every night at midnight
BACKUP_SCHEDULE: "0 0 * * *"

# Every Sunday at 2 AM
BACKUP_SCHEDULE: "0 2 * * 0"
```

## Backup Files

Backups are stored in `./backups/` with naming convention:

```
{container}-{YYYY-MM-DD}T{HH-MM-SS}.{ext}.gz

Examples:
postgres-2025-12-20T03-00-00.sql.gz
mysql-2025-12-20T03-00-00.sql.gz
redis-2025-12-20T03-00-00.rdb.gz
sqlite-2025-12-20T03-00-00.db.gz
```

## Restore Commands

### PostgreSQL

```bash
# Using dockstart CLI (planned)
dockstart restore postgres-2025-12-20T03-00-00.sql.gz

# Manual restore
gunzip -c backups/postgres-2025-12-20T03-00-00.sql.gz | \
  docker exec -i postgres psql -U myuser -d mydb
```

### MySQL

```bash
# Manual restore
gunzip -c backups/mysql-2025-12-20T03-00-00.sql.gz | \
  docker exec -i mysql mysql -u root -p mydb
```

### Redis

```bash
# Manual restore
docker stop redis
gunzip -c backups/redis-2025-12-20T03-00-00.rdb.gz > redis-data/dump.rdb
docker start redis
```

### SQLite

```bash
# Manual restore (stop app first!)
docker stop app
gunzip -c backups/sqlite-2025-12-20T03-00-00.db.gz > data/database.db
docker start app
```

## Manual Backup

Trigger an immediate backup:

```bash
docker exec db-backup /usr/local/bin/backup.sh
```

## View Backup Logs

```bash
docker logs db-backup
```

## Disable Backup Sidecar

Remove from docker-compose.yml or set:

```yaml
db-backup:
  profiles:
    - backup  # Only starts with --profile backup
```

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
                                    │   ./backups/    │
                                    │ (host mount)    │
                                    └─────────────────┘
```

## Implementation Status

- [ ] PostgreSQL backup support
- [ ] MySQL backup support
- [ ] Redis backup support
- [ ] SQLite backup support
- [ ] Restore CLI commands
- [ ] Cloud storage support (S3, GCS)
