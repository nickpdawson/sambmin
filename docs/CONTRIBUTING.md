# Contributing

Sambmin is in beta and contributions are welcome — bug reports, feature requests, and pull requests.

## Bug Reports

Open a GitHub issue with:

1. What you expected to happen
2. What actually happened
3. Steps to reproduce
4. Sambmin version (or commit hash), OS, browser
5. Relevant log output (sanitize any sensitive data like DNs or hostnames)

## Feature Requests

Open a GitHub issue describing:

1. The problem you're trying to solve
2. How you'd expect it to work
3. Any alternatives you've considered

## Pull Requests

### Before you start

- Check existing issues and PRs to avoid duplicate work
- For non-trivial changes, open an issue first to discuss the approach
- Fork the repo and create a feature branch from `main`

### Code Style

**Go (backend)**
- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `fmt.Errorf("operation: %w", err)` for error wrapping
- Table-driven tests in `_test.go` files alongside source
- Structured logging with `slog`

**TypeScript/React (frontend)**
- Functional components, hooks for state
- Use Ant Design components — don't reinvent what AntD provides
- Follow existing patterns in `web/src/pages/` and `web/src/components/`
- Run `npm run lint` before submitting

**Python (scripts)**
- PEP 8 style
- JSON output to stdout, errors to stderr
- No external dependencies — stdlib only
- No interactive prompts — all input via CLI args or stdin

### Testing

- Backend: `cd api && go test ./...` — all tests must pass
- Frontend: `cd web && npx tsc -b` — no type errors
- Add tests for new functionality where practical

### Commit Messages

- Use imperative mood: "Add user search" not "Added user search"
- Keep the subject line under 72 characters
- Reference related issues: "Fix #42: handle expired accounts"

### PR Process

1. Ensure all tests pass
2. Keep PRs focused — one feature or fix per PR
3. Update documentation if your change affects user-facing behavior
4. Describe what changed and why in the PR description

## Development Setup

See [BUILD.md](BUILD.md) for prerequisites and build instructions. The [macOS guide](installation/macos.md) covers running in development mode with mock data.

## Architecture

See [ARCHITECTURE.md](ARCHITECTURE.md) to understand the codebase structure before diving in.

## License

By contributing, you agree that your contributions will be licensed under the [GPLv3](../LICENSE).
