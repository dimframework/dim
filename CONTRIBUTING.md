# Contributing to Dim Framework

Thank you for your interest in contributing to Dim Framework! We welcome contributions from everyone. By participating in this project, you help make it better for the entire community.

## How to Contribute

### 1. Reporting Bugs

If you find a bug, please create a new issue on GitHub. Be sure to include:
- A clear title and description.
- Steps to reproduce the issue.
- Expected behavior vs. actual behavior.
- Your Go version and operating system.
- Code snippets or a minimal reproduction repository.

### 2. Suggesting Enhancements

Have an idea for a new feature? Open an issue tagged as `enhancement` or `feature request`. Describe:
- The problem you want to solve.
- Your proposed solution or API design.
- Alternative solutions you've considered.

### 3. Pull Requests

Pull Requests are welcome! Here's the workflow:

1.  **Fork** the repository and clone it locally.
2.  Create a new **branch** for your feature or fix: `git checkout -b feature/amazing-feature`.
3.  **Implement** your changes.
    - Write clean, idiomatic Go code (follow [Effective Go](https://go.dev/doc/effective_go)).
    - Add **Unit Tests** for your changes. We aim for high test coverage.
    - Ensure all tests pass: `go test ./...`
4.  **Format** your code: `go fmt ./...`.
5.  **Commit** your changes with clear messages.
6.  **Push** to your fork and submit a **Pull Request**.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/dimframework/dim.git
cd dim

# Install dependencies
go mod download

# Run tests
go test ./... -v
```

## Coding Style

- We follow standard Go conventions.
- Use `gofmt` to format your code.
- Run `go vet` to catch common errors.
- Exported functions and types must have documentation comments.

## Questions?

Feel free to open a [Discussion](https://github.com/dimframework/dim/discussions) or ask in an Issue for any questions.

Thank you for contributing!
