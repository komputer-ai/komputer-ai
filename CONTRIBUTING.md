# Contributing to komputer.ai

Thank you for your interest in contributing to komputer.ai! This guide will help you get started.

## Prerequisites

- **Go** 1.24+ (operator, API, CLI)
- **Python** 3.12+ (agent)
- **Docker** (building images)
- **Kubernetes** cluster (testing — minikube, kind, or a remote cluster)
- **Helm** 3.x (deployment)
- **kubectl** configured for your cluster

## Repository Structure

| Component | Language | Path |
|-----------|----------|------|
| komputer-operator | Go | `komputer-operator/` |
| komputer-api | Go | `komputer-api/` |
| komputer-agent | Python | `komputer-agent/` |
| komputer-cli | Go | `komputer-cli/` |

Each component is self-contained with its own `go.mod` or `requirements.txt`.

## Building

### Operator

```bash
cd komputer-operator
make build
```

### API

```bash
cd komputer-api
go build -o komputer-api .
```

### Agent

```bash
cd komputer-agent
pip install -r requirements.txt
```

### CLI

```bash
cd komputer-cli
go build -o komputer-cli .
```

## Running Tests

### Operator (e2e)

```bash
cd komputer-operator
make test        # unit tests
make test-e2e    # end-to-end (requires a running cluster)
```

## Pull Requests

1. Fork the repository and create a feature branch from `main`.
2. Use [conventional commits](https://www.conventionalcommits.org/) for your commit messages:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `refactor:` for code refactoring
   - `test:` for test additions or changes
3. Keep PRs focused — one feature or fix per PR.
4. Ensure your changes build and pass tests before submitting.
5. Update documentation if your change affects user-facing behavior.

## Reporting Issues

- Use [GitHub Issues](https://github.com/komputer-ai/komputer-ai/issues) for bug reports and feature requests.
- Check existing issues before creating a new one.
- For security vulnerabilities, see [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [MIT License](LICENSE).
