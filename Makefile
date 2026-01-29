.PHONY: build run test docker-build docker-run clean help

# Build the Go binary
build:
	go build -o agent ./cmd/agent

# Run the agent locally
run: build
	./agent

# Run tests
test:
	go test -v ./...

# Build Docker image
docker-build:
	docker build -t weather-agent .

# Run Docker container (connects to host Ollama)
docker-run: docker-build
	docker run --rm \
		-e OLLAMA_HOST=http://host.docker.internal:11434 \
		weather-agent

# Run with docker-compose (includes Ollama)
compose-up:
	docker-compose up --build

# Stop docker-compose services
compose-down:
	docker-compose down

# Clean build artifacts
clean:
	rm -f agent
	go clean

# Show help
help:
	@echo "Available targets:"
	@echo "  build         - Build the Go binary"
	@echo "  run           - Build and run the agent locally"
	@echo "  test          - Run Go tests"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container (host Ollama)"
	@echo "  compose-up    - Run with docker-compose (includes Ollama)"
	@echo "  compose-down  - Stop docker-compose services"
	@echo "  clean         - Remove build artifacts"
