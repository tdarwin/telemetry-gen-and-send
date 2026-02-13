# Go Development Guidelines

## General Principles

- Use Go 1.25+ features where appropriate
- Follow standard Go project layout conventions
- Prefer simplicity and clarity over cleverness
- Write idiomatic Go code

## Code Style

- Use `gofmt` for formatting (enforced by CI)
- Follow effective Go guidelines
- Use meaningful variable and function names
- Keep functions focused and small
- Prefer composition over inheritance

## Error Handling

- Always check and handle errors
- Don't use panic for normal error conditions
- Return errors, don't log and continue silently
- Wrap errors with context using `fmt.Errorf` with `%w`

## Performance Considerations

- Profile before optimizing
- Use sync.Pool for frequently allocated objects
- Minimize allocations in hot paths
- Use buffered channels appropriately
- Be mindful of goroutine lifecycle and cleanup

## Testing

- Write unit tests for all packages
- Use table-driven tests where appropriate
- Test error conditions, not just happy paths
- Keep tests focused and independent
- Use meaningful test names that describe what's being tested

## Dependencies

- Minimize external dependencies
- Keep dependencies up to date
- Use go modules for dependency management
- Review dependency licenses

## Documentation

- Write godoc comments for exported functions and types
- Keep comments up to date with code changes
- Document non-obvious behavior
- Include usage examples in godoc where helpful

## Concurrency

- Avoid sharing memory; communicate via channels where possible
- Use context for cancellation and timeouts
- Always close channels when done
- Use sync primitives (Mutex, RWMutex) correctly
- Avoid goroutine leaks

## Project-Specific Guidelines

### OpenTelemetry Usage

- Use official OTel SDK types and protobuf definitions
- Follow OTel semantic conventions
- Use proper instrumentation context propagation

### Configuration

- Use YAML for configuration files
- Support environment variable substitution
- Validate configuration on load
- Provide sensible defaults

### Performance Requirements

- Target 1M+ events/sec per sender instance
- Keep memory usage bounded
- Use efficient protobuf marshaling/unmarshaling
- Implement proper batching and rate limiting
