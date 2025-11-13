# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands

### Standard Builds
```bash
make build                    # Build with embedded Web UI (~55MB)
make build-minimal           # Build without Web UI (~20MB, faster)
make build-linux-docker      # Cross-compile for Linux using Docker
```

### Development Workflow
```bash
make test                    # Run all tests
make test-coverage          # Generate HTML coverage report
make run                    # Run locally with test database
make gen-testdb            # Generate test database for development
```

### Web UI Development
When modifying the Web UI:
```bash
# Option 1: Use Makefile (recommended)
make build-web             # Builds frontend, copies to internal/webui/dist/, and builds web-server

# Option 2: Manual steps
cd web-ui
npm install
npm run build              # Outputs to dist/
cd ..
# CRITICAL: Copy to internal/webui/dist/ (not internal/webui/)
# The Go embed directive reads from internal/webui/dist/*
rm -rf internal/webui/dist
mkdir -p internal/webui/dist
cp -r web-ui/dist/. internal/webui/dist/
# Rebuild Go binary to embed new UI
go build -o build/web-server ./cmd/web-server
```

**Important**: The Go embed configuration in `internal/webui/embed_full.go` uses `//go:embed dist/*`, so files must be copied to `internal/webui/dist/`, NOT to `internal/webui/` root directory.

### Docker
```bash
./build_docker.sh          # Build Docker image (auto-detects proxy settings)
docker-compose up -d       # Run with Web UI and AI support
```

## Architecture Overview

### Three-Layer Architecture

The codebase follows a clean separation of concerns:

```
┌─────────────────────────────────────┐
│   Presentation Layer                │
│   - CLI (cmd/main.go)               │
│   - Web API (internal/api/)         │
│   - REPL (internal/repl/)           │
└─────────────┬───────────────────────┘
              │
┌─────────────▼───────────────────────┐
│   Service Layer                     │
│   (internal/service/)               │
│   - DatabaseService                 │
│   - ScanService                     │
│   - SearchService                   │
│   - StatsService                    │
│   - ExportService                   │
│   - TransformService                │
└─────────────┬───────────────────────┘
              │
┌─────────────▼───────────────────────┐
│   Data Access Layer                 │
│   (internal/db/)                    │
└─────────────────────────────────────┘
```

### Service Layer Pattern

**Critical architectural decision (ADR-002)**: Business logic lives in `internal/service/`, not in CLI commands or API handlers. This enables:
- Code reuse between CLI and Web API
- Easier unit testing with mocks
- Separation of concerns

When adding features:
1. Implement business logic in appropriate service (or create new service)
2. Add CLI command in `cmd/` that calls the service
3. Add API endpoint in `internal/api/` that calls the same service

### Key Components

**Database Layer (`internal/db/`)**
- Wraps RocksDB C++ bindings (grocksdb)
- Handles column family management
- Smart key format detection (string, binary, uint64, .NET ticks)
- Read-only mode support

**GraphChain Agent (`internal/graphchain/` - 19 files)**
- AI-powered natural language query system
- Supports 5 LLM providers: OpenAI, Anthropic Claude, Google Gemini, Azure OpenAI, Ollama (local)
- Features: conversation memory, intent classification, query caching, audit logging
- Security: read-only mode, query complexity limits
- Configuration: `config/graphchain.yaml`

**Transform Engine (`internal/transform/`)**
- Python-based data transformations
- Executes Python expressions on key-value pairs
- Supports both inline expressions and script files
- Dry-run mode for previewing changes

**MCP Server (`internal/mcp/`)**
- Model Context Protocol server for Claude Desktop integration
- Standardized tool interface for AI assistants
- Configuration: `config/mcp-server.yaml`

**Web UI (`web-ui/`)**
- React 19 + TypeScript + Vite
- Zustand for state management
- Built static files are embedded into Go binary using `embed` package
- Location after build: `internal/webui/dist/`

## Build Variants

### Full Build (Default)
- Includes embedded Web UI
- Binary size: ~55MB
- Use case: Development, desktop usage, all-in-one deployments
- Command: `make build`

### Minimal Build
- Excludes Web UI (removes `internal/webui/embed.go`)
- Binary size: ~20MB
- Use case: Server environments, CI/CD, CLI-only usage
- Command: `make build-minimal`

The build system automatically manages this by:
1. Renaming `embed.go` to `embed_full.go` for minimal builds
2. Restoring it for full builds
3. The Go embed directive only activates when the file exists

## Development Practices

### Test-Driven Development (from `.cursor/rules/golang-common.mdc`)
- Write tests before implementation
- All features require passing tests
- Test suite must be green before merge
- Document TDD process in commits

### Go Best Practices
- Use Go 1.22+ features (including new ServeMux patterns)
- Standard library preferred over external frameworks where possible
- Comprehensive error handling with custom error types
- Proper logging throughout
- No TODOs or placeholders in production code

### API Design
- RESTful principles using Gin framework
- Input validation on all endpoints
- Appropriate HTTP status codes
- Consistent JSON response formatting
- Middleware for cross-cutting concerns (CORS, logging, etc.)

### Progressive Refactoring (ADR-003)
The codebase is undergoing gradual architectural improvements:
- Phase-by-phase migration to service layer pattern
- Continuous delivery during refactoring
- Some legacy code may exist alongside new patterns

## Configuration Files

### GraphChain AI Agent (`config/graphchain.yaml`)
Controls LLM provider, model selection, security settings:
```yaml
graphchain:
  llm:
    provider: "ollama"      # openai, anthropic, google, azure, ollama
    model: "llama2"
    api_key: "${OPENAI_API_KEY}"
    base_url: "http://localhost:11434"
  agent:
    max_iterations: 10
    enable_memory: true
  security:
    enable_audit: true
    read_only_mode: false
    max_query_complexity: 10
```

### Environment Variables
```bash
GRAPHCHAIN_LLM_PROVIDER=openai
GRAPHCHAIN_LLM_MODEL=gpt-4
GRAPHCHAIN_API_KEY=sk-...
```

## Key Technologies

- **RocksDB**: C++ library via `github.com/linxGnu/grocksdb v1.10.1`
- **CLI Framework**: `github.com/spf13/cobra v1.9.1`
- **HTTP Framework**: `github.com/gin-gonic/gin v1.11.0`
- **LLM Integration**: `github.com/tmc/langchaingo v0.1.14`
- **MCP Protocol**: `github.com/mark3labs/mcp-go v0.32.0`
- **Frontend**: React 19.1.1 + TypeScript + Vite 7.1.7 + TailwindCSS

## Documentation

- `/README.md` - Main documentation
- `/BUILD.md` - Platform-specific build instructions
- `/DOCKER_DEPLOYMENT.md` - Docker deployment guide
- `/docs/ARCHITECTURE_DECISIONS.md` - Architecture Decision Records
- `/docs/WEB_API_DOCUMENTATION.md` - REST API documentation
- `/docs/MCP_SERVER_README.md` - MCP server guide
- `/internal/service/README.md` - Service layer architecture
- `/scripts/transform/README.md` - Data transformation guide
