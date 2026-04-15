# Contributing to MCPBridge-Go

Thank you for your interest in contributing to MCPBridge-Go! We welcome contributions from everyone. This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions with the community.

## Getting Started

### Prerequisites

- Go 1.26.2 or higher
- Make
- Git

### Development Setup

1. **Fork the repository**

   ```bash
   # Go to https://github.com/dipievil/mcpbridge-go and click "Fork"
   ```

2. **Clone your fork**

   ```bash
   git clone https://github.com/YOUR_USERNAME/mcpbridge-go.git
   cd mcpbridge-go
   ```

3. **Add upstream remote**

   ```bash
   git remote add upstream https://github.com/dipievil/mcpbridge-go.git
   ```

4. **Install dependencies**

   ```bash
   go mod download
   ```

5. **Build and test**

   ```bash
   make build
   go test ./...
   ```

## Making Changes

### 1. Create a Feature Branch

Use descriptive branch names following this pattern:

- Features: `feature/description-of-feature`
- Bugfixes: `fix/description-of-bug`
- Documentation: `docs/description-of-change`
- Chores: `chore/description-of-change`

```bash
git checkout -b feature/my-new-feature
```

### 2. Make Your Changes

- Follow Go idioms and best practices
- Use `gofmt` to format code: `gofmt -w ./...`
- Add comments for exported functions and types
- Keep functions focused and readable
- DRY principle: Don't Repeat Yourself

### 3. Write Tests

For new features or bug fixes:

```bash
# Run tests locally
go test -v -race ./...

# Run with coverage
go test -cover ./...
```

Test files should:

- Be in the same package as the code being tested
- End with `_test.go`
- Use idiomatic Go testing patterns
- Test both happy paths and error cases

### 4. Commit Your Changes

Use conventional commit messages for all commits:

```bash
git add .
git commit -m "feat(bridge): add support for timeout configuration"
```

#### Commit Message Format

**Types:**

- `feat` - A new feature
- `fix` - A bug fix
- `docs` - Documentation only changes
- `style` - Changes that don't affect code meaning (formatting, missing semicolons, etc.)
- `refactor` - A code change that neither fixes a bug nor adds a feature
- `perf` - A code change that improves performance
- `test` - Adding missing tests or correcting existing tests
- `chore` - Changes to build process, dependencies, etc.

**Examples:**

```bash
git commit -m "feat(config): add env_vars configuration option"
git commit -m "fix(bridge): handle missing env file gracefully"
git commit -m "docs(readme): update installation instructions"
git commit -m "test(bridge): add tests for JSON-RPC message handling"
```

## Questions?

- 📖 Check the [README](README.md) first
- 🐛 Search [existing issues](https://github.com/dipievil/mcpbridge-go/issues)
- 💬 Open a [discussion](https://github.com/dipievil/mcpbridge-go/discussions)
- 📧 Contact the maintainers

Thank you for contributing to MCPBridge-Go! 🎉
