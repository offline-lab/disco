# AGENTS.md - Coding Guidelines for NSS Daemon

## Build, Test, and Lint Commands

### Building
```bash
make                  # Build all binaries (nss-daemon, nss-query, nss-status, etc.)
make clean           # Remove build artifacts
make libnss          # Build NSS module (Linux with glibc only)
go build -o nss-daemon cmd/daemon/main.go  # Build single binary
```

### Testing
```bash
make test            # Run all tests with verbose output: go test -v ./...
go test -v ./...     # Run all tests directly
go test -v -run TestRecordStore_AddAndGet ./internal/daemon/  # Run single test
go test -v -run TestRecordStore ./internal/daemon/           # Run tests matching pattern
```

### Integration Testing
```bash
test/nss-test.sh     # Multi-node integration test (requires running daemon)
```

## Code Style Guidelines

### Import Organization
- Standard library imports first (alphabetically sorted)
- Third-party imports second (alphabetically sorted)
- Empty line between groups
- No comments in import blocks

Example:
```go
import (
    "fmt"
    "os"
    "time"

    "gopkg.in/yaml.v3"

    "github.com/flip/nss-daemon/internal/nss"
)
```

### Formatting
- Use `go fmt` conventions (tabs for indentation, no trailing whitespace)
- Struct fields: PascalCase
- Exported functions/types: PascalCase
- Unexported functions/variables: camelCase
- Constants: PascalCase with descriptive prefixes (e.g., `MessageAnnounce`, `keySize`)
- Package names: lowercase, single word

### Error Handling
- Always return errors, never panic in library code
- Use `fmt.Errorf` with `%w` verb for error wrapping
- Check errors immediately after operations
- Functions return `(result, error)` tuples
- Example: `return nil, fmt.Errorf("failed to read config: %w", err)`

### Naming Conventions
- Interfaces: Simple nouns (e.g., `Listener`, `Announcer`)
- Constructors: `NewTypeName` (e.g., `NewRecordStore`, `NewDaemon`)
- Getter methods: `GetField()` for simple access
- Booleans: Prefix with `Is`, `Has`, `Can` (e.g., `isLocalHost`)
- Files: lowercase, single word or underscore-separated

### Concurrency
- Use `sync.RWMutex` for read/write locking on shared state
- Always defer unlock: `defer mu.Unlock()` after lock acquisition
- Always stop tickers: `defer ticker.Stop()`
- Graceful shutdown with `select` on stop channels (see Additional Patterns)

### Testing
- Test files: `*_test.go`
- Test functions: `TestFunctionName`
- Use `t.Fatal()` for test-ending failures
- Use `t.Errorf()` for non-fatal assertions
- Run single test: `go test -v -run TestName ./path/`

### Logging
- Use `internal/logging` package for structured logging
- Log levels: Debug, Info, Warn, Error, Fatal
- Supports text and JSON formats via config
- Always include error in log fields when available
- Example: `logging.Error("Failed to connect", err, map[string]interface{}{"host": host})`

### Comments
- Exported types/functions should have package-level comments
- No inline comments for obvious code
- Comments should explain "why", not "what"

## Project Structure
```
cmd/           - CLI entry points (main.go files)
internal/      - Private packages (config, daemon, discovery, security, nss, service, logging)
libnss/        - C NSS module (Linux only)
test/          - Integration test scripts
docs/          - Documentation
```

## File Formatting
```bash
gofmt -s -w .    # Format all Go files (simplify code)
gofmt -d .        # Show formatting diff without making changes
```

## Additional Patterns

### Config Structs
- Use YAML tags for serialization: `yaml:"field_name"`
- Include `SetDefaults()` method on config structs
- Implement `Validate()` method returning error
- Example validation: check ranges, required fields, file paths

```go
func (c *Config) Validate() error {
    if c.SocketPath == "" {
        return fmt.Errorf("socket_path is required")
    }
    if c.BroadcastInterval < 5*time.Second {
        return fmt.Errorf("broadcast_interval must be at least 5 seconds")
    }
    return nil
}
```

### Stop Channels
- Use `chan struct{}` for stop signals (empty struct = zero bytes)
- Always check stopChan in select statements
- Close stop channel to signal shutdown to all goroutines

### Time Operations
- Use `time.Duration` types with explicit units: `30 * time.Second`
- Store timestamps as `int64` Unix time for JSON compatibility
- Example: `time.Now().Unix()`

### Rate Limiting
- Token bucket algorithm for rate limiting
- Use mutex to protect shared state
- Example: `internal/discovery/ratelimit.go`

```go
type RateLimiter struct {
    maxBurst  int
    rate      int
    tokens    int
    lastTime  time.Time
    mu        sync.Mutex
}

func (rl *RateLimiter) Allow() bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()
    // ... token bucket logic
}
```

## Key Dependencies
- `gopkg.in/yaml.v3` - Configuration parsing
- `golang.org/x/crypto` - Cryptographic functions (indirect)

## Important Notes
- Go 1.22+ required
- No DNS server - uses native NSS integration
- Target platform: Linux with glibc for NSS module
- Daemon runs as systemd service typically
- Use `go fmt` and run `make test` before committing
