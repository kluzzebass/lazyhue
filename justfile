# LazyHue - Philips Hue TUI

# Default recipe: show help
_default:
    @just --list

# Build the binary
build:
    go build -o build/lazyhue ./cmd/lazyhue

# Run the application
run:
    ./build/lazyhue

# Build and run in one step
dev: build run

# Watch for changes and auto-rebuild/restart (requires fswatch)
watch:
    ./watch.sh

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Format code
fmt:
    go fmt ./...

# Tidy go.mod
tidy:
    go mod tidy

# Run go vet
vet:
    go vet ./...

# Clean build artifacts
clean:
    rm -rf build

# Install dependencies
deps:
    go mod download

# Check for issues (fmt, vet, test)
check: fmt vet test
