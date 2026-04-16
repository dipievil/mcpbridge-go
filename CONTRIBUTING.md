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

```
<type>(<scope>): <subject>

<body>

<footer>
```

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

### 5. Keep Your Branch Updated

```bash
git fetch upstream
git rebase upstream/main
```

### 6. Push to Your Fork

```bash
git push origin feature/my-new-feature
```

## Submitting a Pull Request

### Before Submitting

1. **Test your changes locally**
   ```bash
   make build
   go test -v ./...
   go vet ./...
   gofmt -w ./...
   ```

2. **Verify the build passes**
   ```bash
   make clean
   make build
   ```

3. **Update documentation** if needed
   - Update README.md if adding features
   - Add comments to exported functions
   - Update this CONTRIBUTING.md if relevant

### Creating the PR

1. Go to your fork on GitHub
2. Click "New Pull Request"
3. Set base branch to `main`
4. Provide a clear title and description

#### PR Title

Use the same format as commit messages:
```
feat(bridge): add environment variable validation
fix(sse): handle client disconnection gracefully
```

#### PR Description

Include:
- **What**: Clear description of changes
- **Why**: Reason/motivation for these changes
- **How**: How the changes work
- **Testing**: How you tested the changes
- **Checklist**:
  - [ ] Tests added/updated
  - [ ] Documentation updated
  - [ ] Conventional commits used
  - [ ] Code formatted with `gofmt`
  - [ ] All tests passing

Example:
```markdown
## What
Add support for configuring RPC call timeout via YAML config.

## Why
Users need the ability to customize timeout for long-running MCPs.

## How
- Added `timeout` field to MCPConfig struct
- Default to 30 seconds if not specified
- Validate timeout value at startup

## Testing
- Added unit tests for timeout handling
- Manually tested with various timeout values
- Verified graceful timeout handling

## Checklist
- [x] Tests added/updated
- [x] Documentation updated
- [x] Conventional commits used
- [x] Code formatted
- [x] All tests passing
```

### PR Review

- Be responsive to feedback
- Make requested changes in new commits (don't force push during review)
- Ask questions if feedback is unclear
- Be respectful and constructive

## Reporting Issues

### Bug Reports

Include:
- **Clear title** describing the issue
- **Description** of the problem
- **Steps to reproduce** the issue
- **Expected behavior** vs actual behavior
- **Environment**: OS, Go version, MCPBridge version
- **Logs or error messages** if applicable

### Feature Requests

Include:
- **Clear title** of the feature
- **Description** of what you want to achieve
- **Use case** and benefits
- **Potential implementation approach** (optional)
- **Examples** of how you'd use the feature

## Code Review Guidelines

### For Contributors

- Keep PRs focused and reasonably sized
- Respond to review comments promptly
- Be open to feedback
- Ask questions if feedback is unclear

### For Reviewers

- Be respectful and constructive
- Explain the "why" behind feedback
- Acknowledge good work
- Focus on code quality and correctness

## Style Guide

### Go Style

Follow the guidelines in [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments):

- Use `gofmt` for formatting
- Keep lines under 100 characters where reasonable
- Use meaningful variable names
- Add comments for exported types and functions
- Prefer small, focused functions
- Use interfaces for abstraction

### Documentation

- Write clear, concise comments
- Use proper grammar and spelling
- Include examples where helpful
- Keep README.md up to date

## Testing Guidelines

### Writing Tests

```go
func TestBridgeInitialization(t *testing.T) {
    config := MCPConfig{
        Name: "test",
        Port: 3000,
        Command: "echo",
        Args: []string{"test"},
    }
    
    bridge, err := NewBridge(config)
    if err != nil {
        t.Fatalf("failed to create bridge: %v", err)
    }
    defer bridge.Close()
    
    if bridge.config.Name != "test" {
        t.Errorf("expected bridge name 'test', got %s", bridge.config.Name)
    }
}
```

### Test Coverage

Aim for high coverage on:
- Core logic (Bridge, message handling)
- Error cases
- Edge cases

Less critical to test:
- Third-party library wrappers
- OS-specific code (if covered by integration tests)

## Questions?

- 📖 Check the [README](README.md) first
- 🐛 Search [existing issues](https://github.com/dipievil/mcpbridge-go/issues)
- 💬 Open a [discussion](https://github.com/dipievil/mcpbridge-go/discussions)
- 📧 Contact the maintainers

## Recognition

Contributors will be recognized in:
- Release notes
- GitHub contributors page
- README acknowledgments section (for significant contributions)

Thank you for contributing to MCPBridge-Go! 🎉
