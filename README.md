# MCP Domain Checker

A Model Context Protocol (MCP) server that provides domain availability checking functionality using the Namecheap API.

## Features

- Check domain registration status for multiple domains (up to 50 per request)
- Built as an MCP server for integration with Claude and other MCP-compatible clients
- RESTful HTTP interface with Server-Sent Events (SSE) support
- Premium domain pricing information and ICANN fees
- Docker support for easy deployment

## Installation

### Prerequisites

- Go 1.25+
- Docker (optional)
- Namecheap API credentials (required for functionality)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jsgv/mcp-domain-checker.git
cd mcp-domain-checker

# Build the application
go build cmd/app/*.go

# Or use just
just build
```

### Using Docker

```bash
# Build Docker image
docker build -t jsgv/mcp-domain-checker:latest .

# Run container
docker run --rm -p 8080:8080 \
  -e NAMECHEAP_API_USER="your-api-username" \
  -e NAMECHEAP_API_KEY="your-api-key" \
  -e NAMECHEAP_USERNAME="your-username" \
  -e NAMECHEAP_CLIENT_IP="your-whitelisted-ip" \
  jsgv/mcp-domain-checker
```

## Configuration

### Environment Variables

The following environment variables are required for Namecheap API integration:

```bash
NAMECHEAP_API_USER="your-api-username"
NAMECHEAP_API_KEY="your-api-key" 
NAMECHEAP_USERNAME="your-username"
NAMECHEAP_CLIENT_IP="your-whitelisted-ip"
NAMECHEAP_ENDPOINT="https://api.namecheap.com/xml.response"  # or sandbox URL
```

Optional configuration:

```bash
LOG_LEVEL="info"          # debug, info, warn, error, fatal, panic
LOG_FORMAT="production"   # production or development
```

## Usage

### Running the Server

```bash
# Run directly
go run cmd/app/*.go

# Or using just
just run

# Using Docker
just run-docker
```

The server will start on `http://localhost:8080`.

### MCP Tool Usage

The server provides a single MCP tool:

- **Tool Name**: `check_availability_namecheap`
- **Description**: Check domain availability using Namecheap API
- **Parameters**: 
  - `domains` (array of strings): List of domains to check (e.g., `["example.com", "example.org"]`)
  - Maximum 50 domains per request

### Testing with MCP Inspector

```bash
# List available tools
just tools-list

# Or manually
npx @modelcontextprotocol/inspector --cli http://localhost:8080 --transport http --method tools/list
```

## Development

### Commands

```bash
# Build the project
just build

# Run the project
just run

# Lint the code
just lint

# Run tests
just test

# Build Docker image
just build-docker

# Run Docker container
just run-docker
```

### Project Structure

```
├── cmd/app/          # Application entry point
│   ├── config.go     # Configuration and logging setup
│   └── main.go       # Main application server
├── internal/pkg/     # Internal packages
│   ├── namecheap/    # Namecheap API client implementation
│   │   └── namecheap.go      # API service and types
│   └── tools/        # MCP tool implementations
│       ├── domain_checker.go      # Factory for domain tools
│       └── namecheap_handler.go   # MCP handler for Namecheap tool
├── Dockerfile        # Docker configuration
├── justfile          # Task runner configuration
├── go.mod            # Go module definition
├── LICENSE           # MIT License
└── README.md         # This file
```

## License

This project is licensed under the MIT License.
