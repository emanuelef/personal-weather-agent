# Weather Wind Forecast Agent

A Go-based agent that fetches wind forecast data for London Heathrow Airport and uses Ollama for AI-powered analysis.

## Features

- Fetches 15-day wind forecast from Open-Meteo API (no API key required - completely free)
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

## Environment Variables

Copy `.env.example` to `.env` and fill in your secrets and configuration. The `.env` file is ignored by git and should not be committed.

```bash
cp .env.example .env
# Edit .env and set your values
```

## Telegram Integration

To receive the Ollama summary via Telegram, set the following environment variables:

- `TELEGRAM_TOKEN`: Your Telegram bot token
- `TELEGRAM_CHAT_ID`: The chat ID to send messages to

You can use a `.env` file for convenience. Example:

```env
OLLAMA_HOST=http://127.0.0.1:11434
OLLAMA_MODEL=llama3.2:3b
FORECAST_DAYS=15
TELEGRAM_TOKEN=your_telegram_bot_token
TELEGRAM_CHAT_ID=your_telegram_chat_id
```

To run with Docker and .env:

```bash
docker run --rm --network host --env-file .env ghcr.io/emanuelef/test-agent:latest
```

Or set variables directly:

```bash
docker run --rm --network host \
  -e TELEGRAM_TOKEN=your_telegram_bot_token \
  -e TELEGRAM_CHAT_ID=your_telegram_chat_id \
  -e OLLAMA_MODEL=llama3.2:3b \
  ghcr.io/emanuelef/test-agent:latest
```

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

### Run container with Ollama on host

When Ollama is installed directly on the host machine (not in Docker):

```bash
# On Linux (e.g., Oracle VM)
docker run --rm --network host weather-agent

# Alternative on Linux - explicitly set host
docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  weather-agent

# On macOS/Windows (use host.docker.internal)
docker run --rm \
  -e OLLAMA_HOST=http://host.docker.internal:11434 \
  weather-agent
```

**For Oracle Cloud VM with Ollama installed on host:**
The simplest approach is to use `--network host`, which allows the container to access services on the host's localhost:

```bash
docker run --rm --network host weather-agent
```

Or build and run using Make:
```bash
make docker-build
docker run --rm --network host weather-agent
```

### Docker Compose (Ollama in Docker)

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
