---
description: 'Instructions for deps-diff-summary Go project'
applyTo: '**/*.go,**/go.mod,**/go.sum'
---

# Copilot Instructions for deps-diff-summary

This file provides context for AI assistants working in this Go library project. It combines project-specific guidance with idiomatic Go practices based on [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), and [Google's Go Style Guide](https://google.github.io/styleguide/go/).

## Project Overview

**deps-diff-summary** is a Go library module (module: `summary`) for comparing PHP Composer lock files. See README.md and SPECIFICATION.md for full details.

## Build, Test, and Lint

### Using Makefile (Recommended)

```bash
# Display help and available targets
make help

# Build the library
make build

# Run all tests and linting
make test

# Run only Go tests
make test-go

# Run code formatting
make fmt

# Run code linting (golangci-lint)
make test-lint

# Run go vet for suspicious patterns
make vet

# Verify dependencies
make verify-deps
```

### Standard Go Commands (Alternative)

```bash
# Build the library
go build ./...

# Run tests (entire test suite)
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Run tests with coverage
go test -cover ./...

# Format code (required before commits)
go fmt ./...

# Lint with golangci-lint (if installed)
golangci-lint run ./...

# Vet for suspicious code patterns
go vet ./...

# Download and tidy dependencies
go mod download
go mod tidy
```

## Project Structure

The project is organized following standard Go conventions:

- **`cmd/`** – CLI entry points (main packages)
- **`internal/`** – Internal packages not meant for external import
- **`pkg/`** – Public libraries (if applicable)
- **`go.mod`** – Module definition

## Go Development Standards

### General Principles

- Write simple, clear, and idiomatic Go code
- Favor clarity and simplicity over cleverness
- Keep the happy path left-aligned (minimize indentation)
- Return early to reduce nesting; prefer `if condition { return }` to avoid else blocks
- Make the zero value useful
- Write self-documenting code with clear, descriptive names
- Document all exported types, functions, methods, and packages
- Prefer standard library solutions (e.g., `strings.Builder`, `filepath.Join`) over custom implementations
- Avoid emoji in code and comments

### Naming Conventions

#### Packages
- Use lowercase, single-word names
- Avoid underscores, hyphens, or mixedCaps
- Describe what the package provides, not what it contains
- Avoid generic names like `util`, `common`, or `base`
- Package names should be singular, not plural

#### Code Organization
- Organize code by functionality, not by type; group related functions/types in the same package
- Use descriptive file names; prefer `thing.go` over `impl.go`
- Unexported helpers belong in `internal/`; only export what's part of the public API

#### Variables and Functions
- Use camelCase (mixedCaps) rather than underscores
- Exported identifiers start with uppercase (e.g., `ProcessComposer`, `NewDiffer`)
- Unexported identifiers start with lowercase
- Single-letter variables only for very short scopes (like loop indices)
- Avoid stuttering (e.g., prefer `http.Server` over `http.HTTPServer`)

#### Interfaces
- Name with `-er` suffix when appropriate (e.g., `Reader`, `Writer`)
- Single-method interfaces should be named after the method (e.g., `Read` → `Reader`)
- Keep interfaces small and focused

#### Constants
- Use MixedCaps for exported constants, mixedCaps for unexported
- Group related constants using `const` blocks
- Consider using typed constants for better type safety

### Code Style and Formatting

- Always use `gofmt` to format code (required before commits)
- Use `goimports` to manage imports automatically
- Keep line length reasonable for readability
- Add blank lines to separate logical groups of code
- Avoid using emoji in code and comments

### Comments

- Strive for self-documenting code; prefer clear names over comments
- Write comments only when explaining complex logic or non-obvious behavior
- Start comment sentences with the name of the thing being described
- Package comments should start with "Package [name]"
- Use line comments (`//`) for most comments
- Use block comments (`/* */`) sparingly, mainly for package documentation
- Document why, not what (unless what is complex)
- Keep error messages lowercase, no ending punctuation

### Error Handling

- Check errors immediately after function calls (use `if err != nil`)
- Don't ignore errors using `_` unless you have a good reason (document why)
- Wrap errors with context using `fmt.Errorf` with `%w` verb
- Create custom error types when checking for specific errors
- Place error returns as the last return value
- Name error variables `err`
- Use `errors.Is` and `errors.As` for error checking
- Return errors up the stack; only log at the boundary (main)

### Type Safety and Language Features

- Define types to add meaning and type safety
- Use struct tags for JSON, XML, database mappings
- Prefer explicit type conversions
- Check the second return value for type assertions
- Prefer generics over unconstrained types; use `any` (Go 1.18+) only when truly unconstrained
- **CRITICAL: Package Declaration Rules**
  - Each `.go` file must have exactly ONE `package` line
  - When editing existing files, **PRESERVE** the existing `package` declaration
  - When creating new files, check what package name other `.go` files in the same directory use and use the SAME name
  - For new directories, use the directory name as the package name
  - When replacing file content, include only ONE `package` declaration at the top

### Pointers vs Values

- Use pointer receivers for large structs or when you need to modify the receiver
- Use value receivers for small structs and when immutability is desired
- Use pointer parameters when you need to modify the argument or for large structs
- Use value parameters for small structs and to prevent modification
- Be consistent within a type's method set
- Consider the zero value when choosing pointer vs value receivers

### Interfaces and Composition

- Accept interfaces, return concrete types
- Keep interfaces small (1-3 methods is ideal)
- Use embedding for composition
- Define interfaces close to where they're used, not where they're implemented
- Don't export interfaces unless necessary

### Dependency Management

- Keep `go.mod` and `go.sum` in sync; run `go mod tidy` before commits
- Prefer standard library over external dependencies when feasible
- Lock critical dependencies with explicit versions
- Use Go modules for dependency management
- Regularly update dependencies for security patches
- Vendor dependencies only when necessary

## Concurrency

### Goroutines
- Be cautious about creating goroutines in libraries; prefer letting the caller control concurrency
- If you must create goroutines in libraries, provide clear documentation and cleanup mechanisms
- Always know how a goroutine will exit
- Use `sync.WaitGroup` or channels to wait for goroutines
- Avoid goroutine leaks by ensuring cleanup
- **WaitGroup by Go version:**
  - Go 1.25+: Use the new `WaitGroup.Go` method: `wg.Go(task)`
  - Go < 1.25: Use the classic `Add`/`Done` pattern

### Channels
- Use channels to communicate between goroutines
- Don't communicate by sharing memory; share memory by communicating
- Close channels from the sender side, not the receiver
- Use buffered channels when you know the capacity
- Use `select` for non-blocking operations

### Synchronization
- Use `sync.Mutex` for protecting shared state; keep critical sections small
- Use `sync.RWMutex` for many readers, few writers
- Choose between channels (communication) and mutexes (protecting state) based on use case
- Use `sync.Once` for one-time initialization

## API Design

### HTTP Handlers
- Use `http.HandlerFunc` for simple handlers
- Implement `http.Handler` for handlers needing state
- Use middleware for cross-cutting concerns
- Set appropriate status codes and headers
- Handle errors gracefully and return appropriate error responses
- **Router by Go version:**
  - Go 1.22+: Use the enhanced `net/http` `ServeMux` with pattern-based routing and method matching
  - Go < 1.22: Use classic `ServeMux` and handle methods/paths manually (or use a third-party router when justified)

### JSON APIs
- Use struct tags to control JSON marshaling
- Validate input data
- Use pointers for optional fields
- Consider using `json.RawMessage` for delayed parsing
- Handle JSON errors appropriately

### HTTP Clients
- Keep the client struct focused on configuration and dependencies only (base URL, `*http.Client`, auth, headers)
- Do not store per-request state in the client struct
- Do not store or cache `*http.Request` in the client
- Methods should accept `context.Context` and input parameters, assemble the `*http.Request` locally
- Construct a fresh request per method invocation
- Ensure the underlying `*http.Client` is configured (timeouts, transport) and safe for concurrent use
- Always set headers on the request instance, and close response bodies: `defer resp.Body.Close()`
- Handle errors appropriately

## Testing

### Test Organization
- Use `_test` package suffix for black-box testing as most as possible, keep tests in the same package for white-box testing
- Name test files with `_test.go` suffix
- Place test files next to the code they test

### Writing Tests
- Use table-driven tests for multiple test cases
- Name tests descriptively: `Test_functionName_scenario`
- Use subtests with `t.Run` for better organization
- Test both success and error cases
- Mark helper functions with `t.Helper()`
- Use `t.Cleanup()` for resource cleanup
- Create test fixtures for complex setup

## Memory and I/O Performance

### Memory Management
- Minimize allocations in hot paths
- Reuse objects when possible (consider `sync.Pool`)
- Use value receivers for small structs
- Preallocate slices when size is known
- Avoid unnecessary string conversions

### I/O: Readers and Buffers
- Remember: most `io.Reader` streams are consumable once; reading advances state
- To re-read, buffer once with `io.ReadAll`, then create fresh readers via `bytes.NewReader`
- For HTTP requests, keep original payload as `[]byte` and recreate the body: `req.Body = io.NopCloser(bytes.NewReader(buf))`
- Provide `req.GetBody` for redirects/retries: `req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(buf)), nil }`
- Use `io.Pipe` for streaming without buffering entire payload (writes must be sequential)
- For large payloads, avoid unbounded buffering; consider streaming or temporary storage
- Call `(*bufio.Reader).Reset(r)` to reuse a buffered reader with a new underlying reader

## Security Best Practices

- Validate all external input
- Use strong typing to prevent invalid states
- Sanitize data before using in SQL queries
- Be careful with file paths from user input
- Validate and escape data for different contexts (HTML, SQL, shell)
- Use standard library crypto packages; don't implement your own
- Use `crypto/rand` for random number generation
- Store passwords using bcrypt, scrypt, or argon2
- Use TLS for network communication

## Tools and Development Workflow

### Essential Makefile Targets
- `make build`: Build the library
- `make test`: Run all tests and linting (recommended)
- `make test-go`: Run Go tests only
- `make test-lint`: Run golangci-lint
- `make fmt`: Format code with go fmt (required before commits)
- `make vet`: Find suspicious constructs with go vet
- `make verify-deps`: Verify dependencies with go mod verify

### Direct Go Commands (if not using Makefile)
- `go fmt ./...`: Format code (required)
- `go vet ./...`: Find suspicious constructs
- `golangci-lint run ./...`: Additional linting
- `go test ./...`: Run tests
- `go mod verify`: Verify dependencies
- `go mod tidy`: Download and tidy dependencies

### Development Practices
- Use `make test` before committing (runs both tests and linting)
- Use `make fmt` to auto-format code
- Use `make vet` to check for suspicious patterns
- See `make help` for all available targets and options
- Keep commits focused and atomic
- Write meaningful commit messages
- Review diffs before committing

## Common Pitfalls to Avoid

- Not checking errors
- Ignoring race conditions
- Creating goroutine leaks
- Not using defer for cleanup
- Modifying maps concurrently
- Not understanding nil interfaces vs nil pointers
- Forgetting to close resources (files, connections)
- Using global variables unnecessarily
- Over-using unconstrained types; prefer specific types or generics with constraints
- Not considering the zero value of types
- **Creating duplicate `package` declarations** – always check existing files first

## Future Documentation

- **README.md**: Setup, usage, and examples
- **CONTRIBUTING.md**: Contribution guidelines
- **docs/**: Architecture and design decisions (as the project grows)

## Notes

This is a nascent project. As patterns emerge and code grows, this file should be updated to document project-specific conventions beyond these standard Go practices.
