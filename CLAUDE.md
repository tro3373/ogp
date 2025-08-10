# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go CLI tool that extracts OpenGraph metadata from URLs. It can process single URLs from command line arguments or multiple URLs from stdin in parallel.

## Development Commands

### Building
```bash
# Build for current platform
make build

# Build for specific platforms
make build-linux    # Linux AMD64
make build-mac      # Darwin ARM64
make build-windows  # Windows AMD64
make build-android  # Android ARM64

# Clean build artifacts
make clean
```

### Testing
```bash
# Run all tests
make test
# Or directly with go
go test -v ./...

# Run a single test
go test -v ./cmd -run TestHandleUrl
```

### Running the Application
```bash
# Run with test URLs from file
make run

# Run with single URL
go run . https://example.com

# Run with URLs from stdin
cat urls.txt | go run .
```

### Security and Code Quality
```bash
# Run security check with gosec
make sec
# Or directly
gosec --color=false ./...
```

### Dependency Management
```bash
# Show all dependencies
make deps

# Update dependencies
make update

# Tidy go.mod
make tidy

# Tidy with current Go version
make tidy-go
```

### Release Management
```bash
# Create new tag (increments patch version)
make tag

# Push tags
make tagp

# GoReleaser commands
make gr_check       # Check configuration
make gr_snap        # Create snapshot release
make gr_build       # Build snapshot
```

## Architecture

### Project Structure
- **main.go**: Entry point that calls cmd.Execute()
- **cmd/**: Core application logic
  - **root.go**: Cobra command setup and configuration
  - **handler.go**: Main business logic for OpenGraph extraction
  - **task.go**: Worker task structure for concurrent processing
  - **task_result.go**: Result structure for processed URLs
- **test/**: Test fixtures and data

### Key Design Patterns

1. **Concurrent Processing**: Uses goroutines and channels to process multiple URLs in parallel
   - Worker pool pattern with configurable number of workers (MULTI=2)
   - Context-based cancellation for graceful shutdown
   - Channel-based communication between workers and result collector

2. **Command Pattern**: Uses Cobra framework for CLI structure
   - Configuration support via Viper
   - Supports config file at ~/.ogp

3. **Error Handling**: Wrapped errors with context using pkg/errors

### Dependencies
- **github.com/dyatlov/go-opengraph/opengraph**: Core OpenGraph parsing
- **github.com/spf13/cobra**: CLI framework
- **github.com/spf13/viper**: Configuration management
- **github.com/sirupsen/logrus**: Structured logging
- **github.com/pkg/errors**: Error wrapping

## Debugging

Enable debug logging by setting the LOG_LEVEL environment variable:
```bash
LOG_LEVEL=debug go run . https://example.com
```

Or set DEBUG constant to true in cmd/handler.go for additional debug output.

## Output Format

The tool outputs JSON-formatted OpenGraph data:
- Single URL: Returns single OpenGraph object
- Multiple URLs: Returns array of OpenGraph objects