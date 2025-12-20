# ADR-003: Database Backup Sidecar Architecture

**Status**: Accepted
**Date**: 2025-12-20
**Issue**: #36

## Context

Dockstart generates Docker dev environments that include database services (PostgreSQL, Redis, MySQL, SQLite). Developers need automated backups to:
1. Protect dev data from accidental loss
2. Share database state between team members
3. Reset to known states for testing
4. Migrate data between environments

### Problem Statement

Developers need:
1. Automated database backups without manual setup
2. Easy restore workflow for dev environments
3. Reasonable retention without consuming excessive disk space
4. Support for multiple database types in the same project

## Research: Backup Strategies by Database

### PostgreSQL

| Approach | Command | Hot/Cold | Container Size |
|----------|---------|----------|----------------|
| pg_dump | `pg_dump -U user db \| gzip > backup.sql.gz` | Hot | ~8MB |
| pg_dumpall | `pg_dumpall -U user \| gzip > backup.sql.gz` | Hot | ~8MB |
| pg_basebackup | `pg_basebackup -D /backup` | Cold | ~8MB |

**Recommendation**: Use `pg_dump` for logical backups with `--no-owner --clean --if-exists` flags for portability.

### Redis

| Approach | Command | Durability | Size |
|----------|---------|------------|------|
| RDB Snapshot | `redis-cli BGSAVE` | Point-in-time | Compact |
| AOF | Continuous append | High | Larger |
| Combined | RDB + AOF | Best of both | Medium |

**Recommendation**: Use RDB snapshots (default Redis mode) with periodic BGSAVE trigger.

### SQLite

| Approach | Command | Hot/Cold | Notes |
|----------|---------|----------|-------|
| VACUUM INTO | `.backup` or VACUUM INTO | Requires stop | Optimized copy |
| File copy | `cp database.db backup.db` | Requires stop | Risk of WAL corruption |

**Recommendation**: Stop container before backup, use VACUUM INTO for optimized copy.

### MySQL/MariaDB

| Approach | Command | Hot/Cold | Container Size |
|----------|---------|----------|----------------|
| mysqldump | `mysqldump -u root db \| gzip > backup.sql.gz` | Hot | ~12MB |
| mysqlpump | `mysqlpump -u root db \| gzip > backup.sql.gz` | Hot | ~12MB |

**Recommendation**: Use `mysqldump` with `--single-transaction` for InnoDB tables.

## Research: Scheduling Approaches

### Supercronic (Recommended)

```yaml
command: supercronic /etc/crontab
```

**Advantages**:
- Designed for containers
- Preserves environment variables
- Outputs to stdout/stderr (container-native logging)
- Graceful signal handling (SIGTERM)
- Single binary, lightweight (~3MB)
- Crontab-compatible syntax

**Disadvantages**:
- Extra dependency

### Native Cron (Not Recommended)

```yaml
command: crond -f
```

**Disadvantages**:
- Purges environment before jobs
- Email/discard output model
- Poor signal handling
- Requires complex workarounds

### Sleep Loop (Simple Cases)

```yaml
command: |
  while true; do
    /backup.sh
    sleep 86400
  done
```

**Advantages**:
- Ultra-simple, no dependencies
- Works for single-task containers

**Disadvantages**:
- No scheduling precision
- No overlap detection
- Less efficient

## Decision

### Architecture: Unified Backup Sidecar

Use a single lightweight container that handles all database backup types:

```yaml
services:
  db-backup:
    image: dockstart-backup:alpine
    depends_on:
      - postgres  # or mysql, redis
    volumes:
      - ./backups:/backup
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - BACKUP_SCHEDULE=0 3 * * *
      - BACKUP_RETENTION_DAYS=7
      - DB_TYPE=postgres
      - DB_CONTAINER=postgres
      - DB_NAME=${POSTGRES_DB}
      - DB_USER=${POSTGRES_USER}
      - DB_PASSWORD=${POSTGRES_PASSWORD}
    restart: unless-stopped
```

### Container Image: Alpine-Based

Target size: <25MB including all database clients.

```dockerfile
FROM alpine:3.19

RUN apk add --no-cache \
    supercronic \
    postgresql-client \
    mysql-client \
    redis \
    sqlite \
    docker-cli \
    gzip \
    bash

COPY scripts/ /usr/local/bin/
COPY entrypoint.sh /entrypoint.sh

VOLUME ["/backup"]
ENTRYPOINT ["/entrypoint.sh"]
CMD ["supercronic", "/etc/crontab"]
```

### Scheduling: Supercronic

Use Supercronic for its container-native design:

```
# /etc/crontab - Generated at runtime
0 3 * * * /usr/local/bin/backup.sh
0 4 * * * /usr/local/bin/rotate.sh
```

### File Naming Convention

```
{db_container}-{date}T{time}.{ext}.gz

Examples:
postgres-2025-12-20T03-00-00.sql.gz
mysql-2025-12-20T03-00-00.sql.gz
redis-2025-12-20T03-00-00.rdb.gz
sqlite-2025-12-20T03-00-00.db.gz
```

### Rotation Strategy

Default for dev environments:
- **Retention**: 7 days
- **Frequency**: Daily at 3 AM
- **Method**: Delete files older than retention period

```bash
# rotate.sh
find /backup -name "${DB_CONTAINER}-*.gz" -mtime +${BACKUP_RETENTION_DAYS} -delete
```

## Restore Workflow

### PostgreSQL

```bash
# From backup sidecar
dockstart restore postgres --backup postgres-2025-12-20T03-00-00.sql.gz

# Manual equivalent
gunzip -c backups/postgres-2025-12-20.sql.gz | \
  docker exec -i postgres psql -U user -d database
```

### Redis

```bash
# From backup sidecar
dockstart restore redis --backup redis-2025-12-20T03-00-00.rdb.gz

# Manual equivalent
docker stop redis
gunzip -c backups/redis-2025-12-20.rdb.gz > redis-data/dump.rdb
docker start redis
```

### MySQL

```bash
# From backup sidecar
dockstart restore mysql --backup mysql-2025-12-20T03-00-00.sql.gz

# Manual equivalent
gunzip -c backups/mysql-2025-12-20.sql.gz | \
  docker exec -i mysql mysql -u root -p database
```

### SQLite

```bash
# From backup sidecar
dockstart restore sqlite --backup sqlite-2025-12-20T03-00-00.db.gz

# Manual equivalent
docker stop app  # Stop container using SQLite
gunzip -c backups/sqlite-2025-12-20.db.gz > data/database.db
docker start app
```

## Architecture

### Detection Flow

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│ Detect Database │────▶│ Add Backup       │────▶│ Configure       │
│ Services        │     │ Sidecar          │     │ Schedule/Rotate │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                                          │
                                                          ▼
                              ┌─────────────────────────────────────┐
                              │     Generate Backup Scripts         │
                              │  - backup-{dbtype}.sh               │
                              │  - rotate.sh                        │
                              │  - restore-{dbtype}.sh              │
                              └─────────────────────────────────────┘
```

### Generated Docker Compose Structure

```yaml
services:
  app:
    build: ...
    depends_on:
      - postgres

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: myapp_dev
      POSTGRES_USER: myapp
      POSTGRES_PASSWORD: secret
    volumes:
      - postgres-data:/var/lib/postgresql/data

  # Backup sidecar - auto-generated when database detected
  db-backup:
    build:
      context: .devcontainer
      dockerfile: Dockerfile.backup
    depends_on:
      - postgres
    volumes:
      - ./backups:/backup
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - BACKUP_SCHEDULE=0 3 * * *
      - BACKUP_RETENTION_DAYS=7
      - DB_TYPE=postgres
      - DB_CONTAINER=postgres
      - DB_NAME=myapp_dev
      - DB_USER=myapp
      - DB_PASSWORD=secret
    restart: unless-stopped

volumes:
  postgres-data:
```

### Backup Script: PostgreSQL

```bash
#!/bin/bash
# backup-postgres.sh
set -eo pipefail

DATE=$(date +%Y-%m-%dT%H-%M-%S)
BACKUP_FILE="/backup/${DB_CONTAINER}-${DATE}.sql.gz"

echo "[$(date)] Starting PostgreSQL backup..."

PGPASSWORD="${DB_PASSWORD}" pg_dump \
  -h "${DB_CONTAINER}" \
  -U "${DB_USER}" \
  -d "${DB_NAME}" \
  --no-owner \
  --clean \
  --if-exists \
  | gzip -6 > "${BACKUP_FILE}"

echo "[$(date)] Backup completed: ${BACKUP_FILE}"
echo "[$(date)] Size: $(du -h "${BACKUP_FILE}" | cut -f1)"
```

### Backup Script: Redis

```bash
#!/bin/bash
# backup-redis.sh
set -eo pipefail

DATE=$(date +%Y-%m-%dT%H-%M-%S)
BACKUP_FILE="/backup/${DB_CONTAINER}-${DATE}.rdb.gz"

echo "[$(date)] Starting Redis backup..."

# Trigger background save
redis-cli -h "${DB_CONTAINER}" BGSAVE

# Wait for save to complete
while [ "$(redis-cli -h "${DB_CONTAINER}" LASTSAVE)" == "$(redis-cli -h "${DB_CONTAINER}" LASTSAVE)" ]; do
  sleep 1
done

# Copy and compress the RDB file
docker cp "${DB_CONTAINER}:/data/dump.rdb" - | gzip -6 > "${BACKUP_FILE}"

echo "[$(date)] Backup completed: ${BACKUP_FILE}"
```

### Backup Script: SQLite

```bash
#!/bin/bash
# backup-sqlite.sh
set -eo pipefail

DATE=$(date +%Y-%m-%dT%H-%M-%S)
BACKUP_FILE="/backup/${DB_CONTAINER}-${DATE}.db.gz"

echo "[$(date)] Starting SQLite backup..."

# Stop container to ensure consistent backup
if [ "${STOP_CONTAINER}" = "true" ]; then
  echo "[$(date)] Stopping ${APP_CONTAINER} for consistent backup..."
  docker stop "${APP_CONTAINER}"
fi

# Create backup using VACUUM INTO (optimized)
sqlite3 "${DB_PATH}" "VACUUM INTO '/tmp/backup.db'"
gzip -6 -c /tmp/backup.db > "${BACKUP_FILE}"
rm /tmp/backup.db

# Restart container if stopped
if [ "${STOP_CONTAINER}" = "true" ]; then
  docker start "${APP_CONTAINER}"
  echo "[$(date)] Restarted ${APP_CONTAINER}"
fi

echo "[$(date)] Backup completed: ${BACKUP_FILE}"
```

## Configuration Options

### BackupSidecarConfig

```go
type BackupSidecarConfig struct {
    // Database type (postgres, mysql, redis, sqlite)
    DatabaseType string

    // Container name of the database
    DatabaseContainer string

    // Cron schedule (default: "0 3 * * *" = 3 AM daily)
    Schedule string

    // Retention in days (default: 7)
    RetentionDays int

    // Backup directory on host (default: "./backups")
    BackupPath string

    // Whether to stop container for backup (SQLite)
    StopContainer bool

    // Compression level 1-9 (default: 6)
    CompressionLevel int
}
```

### Detection Rules

| Database | Detection | Auto-Add Backup |
|----------|-----------|-----------------|
| PostgreSQL | `postgres` service detected | Yes |
| MySQL | `mysql` or `mariadb` service | Yes |
| Redis | `redis` service + persistence | Yes |
| SQLite | `.db` file in volumes | Optional |

## Docker Concepts Reference

### Docker Socket Access

Required for container stop/start operations (SQLite backups):

```yaml
volumes:
  - /var/run/docker.sock:/var/run/docker.sock:ro
```

**Security Note**: Read-only access limits risk but still requires trust.

### Named Volumes vs Bind Mounts

**Backups use bind mount** (`./backups:/backup`):
- Directly accessible from host
- Easy to browse, copy, share
- Git-ignorable (add `backups/` to `.gitignore`)

**Database data uses named volume** (`postgres-data:/var/lib/postgresql/data`):
- Managed by Docker
- Better performance on macOS
- Portable between systems

### Healthcheck for Backup Container

```yaml
healthcheck:
  test: ["CMD", "pgrep", "-f", "supercronic"]
  interval: 60s
  timeout: 10s
  retries: 3
```

## Implementation Plan

### Phase 1: Core Backup Infrastructure
- [ ] Create `BackupSidecarConfig` in models (#37)
- [ ] Implement database backup detection rules (#38)
- [ ] Create backup sidecar Dockerfile template (#39)

### Phase 2: Backup Script Generation
- [ ] Generate database-specific backup scripts (#40)
- [ ] Implement rotation script (#41)
- [ ] Create restore helpers (#42)

### Phase 3: Integration
- [ ] Update compose generator for backup sidecar (#43)
- [ ] Add backup configuration to devcontainer.json (#44)
- [ ] Create documentation and examples (#45)

## Consequences

### Positive
- Automated backups without developer configuration
- Consistent backup format across projects
- Easy restore workflow for dev environments
- Lightweight container (~25MB)
- Works with all major databases

### Negative
- Additional container resource usage
- Docker socket access for SQLite backups
- Disk space for backup retention
- More complex generated docker-compose

### Neutral
- Backups stored on host filesystem (by design for dev)
- No cloud storage support initially (can be added later)
- Fixed retention policy (configurable via env vars)

## Alternatives Considered

### 1. Host-Based Cron

Run backups via host cron instead of container.

**Rejected**: Requires host configuration, not portable, breaks containerization.

### 2. Application-Level Backups

Implement backup logic in application code.

**Rejected**: Language-specific, not universal, adds app complexity.

### 3. Existing Tools (docker-volume-backup, kartoza/pg-backup)

Use existing backup containers.

**Rejected**: Too heavy for dev environments, complex configuration, not optimized for our use case.

### 4. No Automatic Backups

Let developers configure their own backups.

**Rejected**: Poor developer experience, inconsistent across projects.

## References

- [Supercronic - Cron for Containers](https://github.com/aptible/supercronic)
- [PostgreSQL pg_dump Documentation](https://www.postgresql.org/docs/current/app-pgdump.html)
- [Redis Persistence](https://redis.io/docs/latest/operate/oss_and_stack/management/persistence/)
- [SQLite Backup Methods](https://sqlite.org/backup.html)
- [Docker Volume Backup](https://github.com/offen/docker-volume-backup)
- [Alpine Docker Images](https://hub.docker.com/_/alpine)
