# Contributing to db-mcp

Thank you for your interest in contributing to db-mcp! This document provides guidelines for contributing to the project.

## Reporting Issues

Please report bugs or request features by opening a [GitHub Issue](https://github.com/your-org/db-mcp/issues).

When reporting a bug, please include:
- A clear description of the problem
- Steps to reproduce the issue
- Expected behavior vs. actual behavior
- Environment details (OS, Go version, database type/version)
- Relevant logs or error messages

## Development Setup

### Prerequisites

- Go 1.26 or higher
- MySQL 5.7+/8.0+ (for MySQL-related development)
- MongoDB 4.0+ (for MongoDB-related development)

### Build

```bash
make build
```

The binary will be built to `./bin/db-mcp`.

### Configuration

Create a `config.yaml` in the project root or set environment variables as described in [README.md](README.md).

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests only
make test-integration

# Generate coverage report
make test-coverage
```

## Code Style

We use `golangci-lint` for code linting:

```bash
make lint
```

Please ensure your code passes linting before submitting a pull request.

## Pull Request Process

1. Fork the repository
2. Create a new branch for your feature or bugfix: `git checkout -b feature/my-feature` or `git checkout -b fix/my-bugfix`
3. Make your changes and commit them
4. Push to your fork: `git push origin feature/my-feature`
5. Open a pull request against the `main` branch

### Commit Message Conventions

We follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages:

- `feat:` - A new feature
- `fix:` - A bug fix
- `docs:` - Documentation changes
- `style:` - Code style changes (formatting, etc.)
- `refactor:` - Code refactoring
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Examples:
- `feat: add support for PostgreSQL`
- `fix: handle null values in batch insert`
- `docs: update API documentation`
- `test: add integration tests for transaction`

## Development Guidelines

- Write unit tests for new functionality
- Ensure all existing tests pass before submitting
- Follow the existing code structure and patterns
- Update documentation when changing behavior
- Add comments for complex logic or non-obvious implementations

## License

By contributing to db-mcp, you agree that your contributions will be licensed under the MIT License.
