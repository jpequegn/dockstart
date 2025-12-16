# dockstart

A CLI tool that analyzes a project and generates Docker development environment files.

**Learning project** for Docker (Dockerfile fundamentals + Compose) and Go.

## Usage

```bash
dockstart ./my-project
```

Generates:
- `.devcontainer/devcontainer.json`
- `.devcontainer/docker-compose.yml`
- `.devcontainer/Dockerfile`

## Supported Languages

- Node.js (package.json)
- Go (go.mod)
- Python (pyproject.toml, requirements.txt)
- Rust (Cargo.toml)

## Detected Services

- PostgreSQL
- Redis

## Development

```bash
# Build locally
go build -o dockstart ./cmd/dockstart

# Build with Docker
docker build -t dockstart .

# Run
./dockstart ./path/to/project
```

## Project Structure

```
dockstart/
├── cmd/dockstart/      # CLI entry point
├── internal/
│   ├── detector/       # Language detection
│   ├── generator/      # File generation
│   └── models/         # Data structures
├── templates/          # Output templates
└── Dockerfile          # Container build
```

## License

MIT
