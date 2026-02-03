# Contributing to tooldiscovery

Thank you for your interest in contributing to tooldiscovery.

## Development Setup

### Prerequisites

- Go 1.25 or later
- golangci-lint (for linting)
- gosec (for security scanning)

### Clone and Build

```bash
git clone https://github.com/jonwraymond/tooldiscovery.git
cd tooldiscovery
go mod download
go build ./...
```

## Testing

### Run All Tests

```bash
go test ./...
```

### Run Tests with Race Detection

```bash
go test -race ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
go test -bench=. ./index
go test -bench=. ./semantic
go test -bench=. ./tooldoc
```

## Code Quality

### Linting

We use golangci-lint with the configuration in `.golangci.yml`:

```bash
golangci-lint run
```

### Security Scanning

```bash
gosec ./...
```

### Vulnerability Checking

```bash
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Commit messages are validated by commitlint in CI.

### Format

```
<type>(<scope>): <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Code style (formatting, semicolons, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvement
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

## Pull Request Process

1. **Fork the repository** and create your branch from `main`.
2. **Write tests** for any new functionality. Maintain or improve test coverage.
3. **Run the full test suite** to ensure your changes don't break existing functionality:
   ```bash
   go test -race ./...
   golangci-lint run
   ```
4. **Update documentation** if you're changing public APIs or behavior.
5. **Use conventional commit messages** for your commits.
6. **Submit your PR** with a clear description of the changes.

## Code Style

- Follow standard Go conventions and `gofmt` formatting
- Add doc comments to all exported types and functions
- Keep interfaces small and focused
- Ensure search ordering is deterministic

## Package Guidelines

### index

- Tool IDs must be normalized to `namespace:name:version`, `namespace:name`, or `name`
- Search results must be deterministic for stable pagination
- OnChange listeners must run outside locks

### search

- BM25 ordering must be deterministic
- Searchers must be safe for concurrent use
- Cache invalidation must be explicit and correct

### semantic

- Strategies must be stateless and concurrency-safe
- Embedders should be deterministic for equal input
- Vector stores must return results ordered by similarity

### tooldoc

- Enforce example size/depth caps to avoid context bloat
- Validate and truncate user-provided documentation fields

### discovery

- Provide composable defaults with explicit options
- Preserve score provenance (BM25 vs embedding vs hybrid)

## Questions?

Open an issue for questions or discussions about contributions.
