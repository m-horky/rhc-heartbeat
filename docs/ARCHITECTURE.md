# Architecture

## Structure

- `cmd/` is the composition root, handling command-line and signal setup, coordinating heartbeat collection, configuration, caching, and processing.
- `pkg/` provide focused APIs and reusable behavior. Package constructors may supply convenient defaults for their own implementation details.
- `internal/` are low-level implementations. They rely on dependency injection from `pkg/`.
