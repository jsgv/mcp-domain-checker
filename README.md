# MCP Domain Checker

A Model Context Protocol (MCP) server that provides domain availability checking functionality using the Namecheap API.

## Features

- Check domain registration status for multiple domains (up to 50 per request)
- Built as an MCP server for integration with Claude and other MCP-compatible clients
- Two transports: **stdio** (spawned by MCP clients locally) and **streamable HTTP** (long-lived service)
- Premium domain pricing information and ICANN fees
- Docker support for easy HTTP deployment

## Installation

### `go install`

```bash
go install github.com/jsgv/mcp-domain-checker/cmd/mcp-domain-checker@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/jsgv/mcp-domain-checker/releases):

```bash
# Using gh CLI (recommended)
gh release download --repo jsgv/mcp-domain-checker --pattern '*linux_amd64*'
tar xzf mcp-domain-checker_*_linux_amd64.tar.gz

# Check version
./mcp-domain-checker --version
```

### Prerequisites

- Go 1.25+ (for building from source)
- Docker (optional)
- Namecheap API credentials (required for functionality)

### Building from Source

```bash
# Clone the repository
git clone https://github.com/jsgv/mcp-domain-checker.git
cd mcp-domain-checker

# Build the application
just build

# Or manually
go build -o mcp-domain-checker ./cmd/mcp-domain-checker
```

### Using Docker

```bash
# Build Docker image
docker build -t jsgv/mcp-domain-checker:latest .

# Run container (HTTP transport on :8080)
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
TRANSPORT="http"          # http or stdio (default: http)
```

## Usage

### Transports

The server supports two transports, selected by the `-transport` flag or the
`TRANSPORT` environment variable (flag wins). Default is `http`.

- `http` — long-lived streamable HTTP server on `:8080`. Use for Docker or
  remote deployments.
- `stdio` — communicates over stdin/stdout using newline-delimited JSON.
  Use when your MCP client spawns the binary as a subprocess. Logs are written
  to stderr so they don't corrupt the protocol framing.

### Running the Server

```bash
# HTTP (default)
just run

# Stdio
just run-stdio

# Docker (HTTP only)
just run-docker
```

The HTTP server listens on `http://localhost:8080`.

### Using with an MCP client (stdio)

After `go install` (or using a downloaded release binary), configure your MCP
client to spawn the binary. Example Claude Desktop config:

```json
{
  "mcpServers": {
    "domain-checker": {
      "command": "mcp-domain-checker",
      "args": ["-transport", "stdio"],
      "env": {
        "NAMECHEAP_API_USER": "your-api-username",
        "NAMECHEAP_API_KEY": "your-api-key",
        "NAMECHEAP_USERNAME": "your-username",
        "NAMECHEAP_CLIENT_IP": "your-whitelisted-ip"
      }
    }
  }
}
```

### Command Line Options

```bash
# Show version
mcp-domain-checker --version
mcp-domain-checker -v

# Select transport (overrides TRANSPORT env)
mcp-domain-checker -transport stdio
mcp-domain-checker -transport http
```

### MCP Tool Usage

The server provides a single MCP tool:

- **Tool Name**: `check_availability_namecheap`
- **Description**: Check domain availability using Namecheap API
- **Parameters**:
  - `domains` (array of strings): List of domains to check (e.g., `["example.com", "example.org"]`)
  - Maximum 50 domains per request

### Testing with MCP Inspector

```bash
# HTTP — requires `just run` in another terminal
just tools-list

# Stdio — requires `just build` first
just tools-list-stdio

# Or manually (HTTP)
npx @modelcontextprotocol/inspector --cli http://localhost:8080 --transport http --method tools/list

# Or manually (stdio)
npx @modelcontextprotocol/inspector --cli ./mcp-domain-checker -transport stdio --method tools/list
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

# Check for dead code
just deadcode

# Build Docker image
just build-docker

# Run Docker container
just run-docker
```

### Project Structure

```
├── cmd/mcp-domain-checker/ # Application entry point
│   ├── config.go         # Configuration and logging setup
│   └── main.go           # Main application server
├── internal/pkg/         # Internal packages
│   ├── namecheap/        # Namecheap API client
│   │   └── namecheap.go  # API service and types
│   └── tool/             # Generic MCP tool wrapper
│       └── tool.go       # Tool implementation
├── .github/workflows/    # CI/CD workflows
├── Dockerfile            # Docker configuration
├── justfile              # Task runner configuration
├── go.mod                # Go module definition
├── LICENSE               # MIT License
└── README.md             # This file
```

## License

This project is licensed under the MIT License.
