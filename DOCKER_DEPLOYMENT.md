# RocksDB CLI Docker Deployment Guide

This guide explains how to deploy RocksDB CLI with web UI and AI support using Docker.

## Quick Start

### 1. Using Docker Compose (Recommended)

```bash
# Copy environment template
cp .env.example .env

# Edit .env and add your API keys
nano .env

# Start the service
docker-compose up -d

# View logs
docker-compose logs -f

# Access the web UI
open http://localhost:8090
```

### 2. Using Docker Run

```bash
docker build -f Dockerfile.web -t rocksdb-cli-web .

docker run -d \
  --name rocksdb-cli-web \
  -p 8090:8090 \
  -v $(pwd)/data:/data \
  -e GRAPHCHAIN_LLM_PROVIDER=openai \
  -e GRAPHCHAIN_LLM_MODEL=gpt-4 \
  -e GRAPHCHAIN_API_KEY=your-api-key \
  rocksdb-cli-web
```

## Configuration

### Environment Variables

#### AI/LLM Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `GRAPHCHAIN_LLM_PROVIDER` | LLM provider (openai, anthropic, azureopenai, google, ollama) | openai | Yes |
| `GRAPHCHAIN_LLM_MODEL` | Model name | gpt-4 | Yes |
| `GRAPHCHAIN_API_KEY` | API key for the provider | - | Yes (except Ollama) |
| `GRAPHCHAIN_BASE_URL` | Custom API endpoint | - | No |

#### Azure OpenAI (Optional)

| Variable | Description | Required for Azure |
|----------|-------------|-------------------|
| `GRAPHCHAIN_AZURE_ENDPOINT` | Azure endpoint URL | Yes |
| `GRAPHCHAIN_AZURE_DEPLOYMENT` | Deployment name | Yes |
| `GRAPHCHAIN_AZURE_API_VERSION` | API version | No (default: 2024-02-01) |

#### Security

| Variable | Description | Default |
|----------|-------------|---------|
| `GRAPHCHAIN_READ_ONLY` | Enable read-only mode | false |
| `GRAPHCHAIN_ENABLE_AUDIT` | Enable audit logging | true |

#### Server

| Variable | Description | Default |
|----------|-------------|---------|
| `WEB_PORT` | Web UI port | 8090 |
| `DB_PATH` | RocksDB database path | ./data |

## AI Provider Examples

### OpenAI

```bash
# .env file
GRAPHCHAIN_LLM_PROVIDER=openai
GRAPHCHAIN_LLM_MODEL=gpt-4
GRAPHCHAIN_API_KEY=sk-...
```

### Anthropic Claude

```bash
# .env file
GRAPHCHAIN_LLM_PROVIDER=anthropic
GRAPHCHAIN_LLM_MODEL=claude-3-opus-20240229
GRAPHCHAIN_API_KEY=sk-ant-...
```

### Azure OpenAI

```bash
# .env file
GRAPHCHAIN_LLM_PROVIDER=azureopenai
GRAPHCHAIN_LLM_MODEL=gpt-4
GRAPHCHAIN_API_KEY=your-azure-key
GRAPHCHAIN_AZURE_ENDPOINT=https://your-resource.openai.azure.com
GRAPHCHAIN_AZURE_DEPLOYMENT=your-deployment-name
```

### Google Gemini

```bash
# .env file
GRAPHCHAIN_LLM_PROVIDER=google
GRAPHCHAIN_LLM_MODEL=gemini-pro
GRAPHCHAIN_API_KEY=your-google-api-key
```

### Ollama (Local, No API Key Required)

```bash
# Start Ollama service with docker-compose
docker-compose --profile ollama up -d

# Pull a model
docker exec ollama ollama pull llama2

# Configure to use Ollama
# .env file
GRAPHCHAIN_LLM_PROVIDER=ollama
GRAPHCHAIN_LLM_MODEL=llama2
GRAPHCHAIN_BASE_URL=http://ollama:11434
```

## Usage Examples

### 1. Start with OpenAI

```bash
# Create .env
cat > .env <<EOF
GRAPHCHAIN_LLM_PROVIDER=openai
GRAPHCHAIN_LLM_MODEL=gpt-4
GRAPHCHAIN_API_KEY=sk-your-openai-key
WEB_PORT=8090
DB_PATH=./my-database
EOF

# Start
docker-compose up -d

# Check logs
docker-compose logs -f rocksdb-cli-web
```

### 2. Start with Local Ollama

```bash
# Create .env
cat > .env <<EOF
GRAPHCHAIN_LLM_PROVIDER=ollama
GRAPHCHAIN_LLM_MODEL=llama2
GRAPHCHAIN_BASE_URL=http://ollama:11434
WEB_PORT=8090
EOF

# Start with Ollama profile
docker-compose --profile ollama up -d

# Pull model (first time only)
docker exec ollama ollama pull llama2

# Access web UI
open http://localhost:8090
```

### 3. Production Deployment with Azure OpenAI

```bash
# Create .env for production
cat > .env <<EOF
GRAPHCHAIN_LLM_PROVIDER=azureopenai
GRAPHCHAIN_LLM_MODEL=gpt-4
GRAPHCHAIN_API_KEY=${AZURE_API_KEY}
GRAPHCHAIN_AZURE_ENDPOINT=https://my-company.openai.azure.com
GRAPHCHAIN_AZURE_DEPLOYMENT=gpt-4-production
GRAPHCHAIN_READ_ONLY=true
GRAPHCHAIN_ENABLE_AUDIT=true
WEB_PORT=8090
DB_PATH=/mnt/production-db
EOF

# Start
docker-compose up -d
```

## Advanced Configuration

### Custom Configuration File

You can mount a custom configuration file:

```yaml
# docker-compose.override.yml
services:
  rocksdb-cli-web:
    volumes:
      - ./my-custom-config.yaml:/etc/rocksdb-cli/config/graphchain.yaml:ro
```

### Multiple Instances

Run multiple instances on different ports:

```bash
# Instance 1 (OpenAI)
docker run -d --name rocksdb-web-openai \
  -p 8090:8090 \
  -v $(pwd)/data1:/data \
  -e GRAPHCHAIN_LLM_PROVIDER=openai \
  -e GRAPHCHAIN_API_KEY=sk-... \
  rocksdb-cli-web

# Instance 2 (Claude)
docker run -d --name rocksdb-web-claude \
  -p 8091:8090 \
  -v $(pwd)/data2:/data \
  -e GRAPHCHAIN_LLM_PROVIDER=anthropic \
  -e GRAPHCHAIN_API_KEY=sk-ant-... \
  rocksdb-cli-web
```

## Health Checks

The container includes a health check endpoint:

```bash
# Check container health
docker-compose ps

# Manual health check
curl http://localhost:8090/health
```

## Troubleshooting

### AI Not Working

1. Check environment variables:
```bash
docker-compose exec rocksdb-cli-web env | grep GRAPHCHAIN
```

2. Check logs for AI errors:
```bash
docker-compose logs rocksdb-cli-web | grep -i "llm\|ai\|error"
```

3. Verify API key is set:
```bash
docker-compose exec rocksdb-cli-web sh -c 'echo $GRAPHCHAIN_API_KEY'
```

### Web UI Not Accessible

1. Check if container is running:
```bash
docker-compose ps
```

2. Check port binding:
```bash
docker-compose port rocksdb-cli-web 8090
```

3. Check logs:
```bash
docker-compose logs rocksdb-cli-web
```

### Database Access Issues

1. Check volume mount:
```bash
docker-compose exec rocksdb-cli-web ls -la /data
```

2. Check permissions:
```bash
ls -la ./data
```

## Stopping and Cleanup

```bash
# Stop services
docker-compose down

# Stop and remove volumes
docker-compose down -v

# Remove all
docker-compose down -v --rmi all
```

## Building from Source

### Build Base Image (One Time)

```bash
# This takes ~10 minutes
docker build -f Dockerfile.base -t rocksdb-base:10.2.1 .
```

### Build Web Image

```bash
# Fast build using base image
docker build -f Dockerfile.web -t rocksdb-cli-web .
```

## Production Recommendations

1. **Use .env file**: Never commit .env with real credentials
2. **Enable read-only mode**: Set `GRAPHCHAIN_READ_ONLY=true` for production
3. **Enable audit logging**: Set `GRAPHCHAIN_ENABLE_AUDIT=true`
4. **Use volume for data**: Always mount a persistent volume for `/data`
5. **Monitor health**: Use health check endpoint for monitoring
6. **Resource limits**: Set memory/CPU limits in docker-compose.yml
7. **Network security**: Use reverse proxy (nginx) with SSL/TLS
8. **Backup**: Regularly backup the `/data` directory

## Example Production docker-compose.yml

```yaml
version: '3.8'

services:
  rocksdb-cli-web:
    build:
      context: .
      dockerfile: Dockerfile.web
    image: rocksdb-cli-web:latest
    container_name: rocksdb-cli-prod
    ports:
      - "127.0.0.1:8090:8090"  # Only localhost
    volumes:
      - /mnt/production/db:/data
      - ./config:/etc/rocksdb-cli/config:ro
    environment:
      - GRAPHCHAIN_LLM_PROVIDER=${GRAPHCHAIN_LLM_PROVIDER}
      - GRAPHCHAIN_LLM_MODEL=${GRAPHCHAIN_LLM_MODEL}
      - GRAPHCHAIN_API_KEY=${GRAPHCHAIN_API_KEY}
      - GRAPHCHAIN_READ_ONLY=true
      - GRAPHCHAIN_ENABLE_AUDIT=true
    restart: always
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
        reservations:
          cpus: '1'
          memory: 2G
    healthcheck:
      test: ["CMD", "/usr/local/bin/rocksdb-cli", "healthcheck"]
      interval: 30s
      timeout: 3s
      retries: 3
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

## Support

For issues and questions:
- GitHub Issues: https://github.com/YOUR_ORG/rocksdb-cli/issues
- Documentation: See README.md for full documentation
