# Log Aggregator Example

This example demonstrates dockstart's automatic log aggregator sidecar generation for a Node.js application using [pino](https://github.com/pinojs/pino) for structured JSON logging.

## What This Shows

When you run `dockstart` on this project, it will:

1. Detect Node.js 20 from `package.json`
2. Detect `pino` as a structured logging library
3. Infer JSON log format
4. Generate a Fluent Bit sidecar configuration

## Quick Start

```bash
# Generate devcontainer files
cd docs/examples/log-aggregator
dockstart .

# Or preview without writing
dockstart --dry-run .
```

## Expected Output

```
📂 Analyzing .
🔍 Detecting project configuration...
   ✅ Detected: node 20 (confidence: 100%)
   📦 Services: []
   📋 Logging: [pino] (JSON format)

📝 Generating devcontainer.json...
📝 Generating docker-compose.yml...
📝 Generating Dockerfile...
📝 Generating fluent-bit.conf...

✨ Done!
```

## Generated Files

After running dockstart, you'll have:

```
.devcontainer/
├── devcontainer.json     # VS Code configuration
├── docker-compose.yml    # App + Fluent Bit services
├── Dockerfile            # Node.js development image
└── fluent-bit.conf       # Log collection config
```

## Using the Dev Container

1. Open the project in VS Code
2. Click "Reopen in Container" when prompted
3. The app and Fluent Bit will start automatically

## Testing the Logging

Once inside the container:

```bash
# Start the app
npm start

# In another terminal, make requests
curl http://localhost:3000/
curl http://localhost:3000/health
curl -X POST http://localhost:3000/users \
  -H "Content-Type: application/json" \
  -d '{"name": "Test", "email": "test@example.com"}'
curl http://localhost:3000/error

# View processed logs
docker compose logs -f fluent-bit
```

## Log Output Format

The Fluent Bit sidecar enriches logs with metadata:

```json
{
  "date": 1703001234.567,
  "environment": "development",
  "project": "log-aggregator-example",
  "level": "info",
  "msg": "Server started",
  "port": 3000
}
```

## Customization

After generation, you can modify `.devcontainer/fluent-bit.conf` to:

- Add Elasticsearch or Loki outputs
- Create custom filters
- Modify log enrichment

See [Log Aggregator Documentation](../../sidecars/log-aggregator.md) for details.
