# Weather Wind Forecast Agent

A Go-based agent that fetches wind forecast data for London Heathrow Airport and uses Ollama for AI-powered analysis.

## Features

- Fetches 15-day wind forecast from Open-Meteo API
- Analyzes wind patterns using local Ollama LLM
- Provides actionable insights for airport operations
- Containerized for easy deployment

## Prerequisites

- Go 1.25 or later (for local development)
- Docker (for containerized deployment)
- Ollama running locally on port 11434

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `OLLAMA_HOST` | `http://127.0.0.1:11434` | Ollama API endpoint |
| `OLLAMA_MODEL` | `llama3.1` | Ollama model to use |
| `FORECAST_DAYS` | `15` | Number of forecast days (max 16) |

## Local Development

```bash
# Run directly
go run ./cmd/agent

# Build binary
go build -o agent ./cmd/agent
./agent

# With custom settings
FORECAST_DAYS=10 OLLAMA_MODEL=llama2 go run ./cmd/agent
```

## Docker Deployment

### Build locally

```bash
docker build -t weather-agent .
```

### Run container

The agent connects to Ollama on the host machine:

```bash
# On Linux
docker run --rm --network host weather-agent

# On macOS/Windows (use host.docker.internal)
docker run --rm \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  weather-agent
```

### Docker Compose (recommended)

If running Ollama in a container:

```yaml
version: '3.8'
services:
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    
  weather-agent:
    image: ghcr.io/emanuelefumagalli/test-agent:latest
    depends_on:
      - ollama
    environment:
      - OLLAMA_HOST=http://ollama:11434
      - OLLAMA_MODEL=llama3.1
      - FORECAST_DAYS=15

volumes:
  ollama-data:
```

## CI/CD

GitHub Actions workflow automatically builds and pushes Docker images to GitHub Container Registry on:
- Push to main/master branch
- Tagged releases
- Pull requests (build only)

Access images at: `ghcr.io/emanuelefumagalli/test-agent:latest`

## Example Output

```
15-day London Heathrow wind forecast (km/h):
Date        | Wind Max | Gust Max
------------+----------+---------
2026-01-29 |     18.5 |    32.1
2026-01-30 |     22.3 |    38.5
...

Ollama summary:
The wind forecast for London Heathrow shows moderate conditions for the next 15 days...
```

## License

MIT
