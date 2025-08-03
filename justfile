GIT_HASH := `git rev-parse --short HEAD`
IMAGE_NAME := "jsgv/mcp-domain-checker"
IMAGE_TAG_HASH := IMAGE_NAME + ":" + GIT_HASH
IMAGE_TAG_LATEST := IMAGE_NAME + ":latest"

# Set environment variables for the application
export LOG_LEVEL           := "DEBUG"
export NAMECHEAP_API_USER  := ""
export NAMECHEAP_API_KEY   := ""
export NAMECHEAP_USERNAME  := ""
export NAMECHEAP_CLIENT_IP := ""
export NAMECHEAP_ENDPOINT  := "https://api.namecheap.com/xml.response"

build:
    go build cmd/app/*.go

run:
    go run cmd/app/*.go

lint:
    golangci-lint run --config .golangci.yaml

build-docker:
    docker build -t {{ IMAGE_TAG_HASH }} -t {{ IMAGE_TAG_LATEST }} .

run-docker:
    docker run --rm -p 8080:8080 {{ IMAGE_TAG_LATEST }}

tools-list:
    npx @modelcontextprotocol/inspector --cli http://localhost:8080/sse --transport http --method tools/list
