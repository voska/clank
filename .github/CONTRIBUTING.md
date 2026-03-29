# Contributing to clank

## Development setup

```bash
git clone https://github.com/voska/clank.git
cd clank
go mod download
make build
```

Requirements: Go 1.22+, tmux 3.2+

## Running tests

```bash
go test ./...
go test -race ./...
```

## Running the linter

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
golangci-lint run
```

## Making changes

1. Fork the repository
2. Create a branch: `git checkout -b feat/your-feature`
3. Make your changes
4. Run tests and lint
5. Commit with a conventional commit message (see below)
6. Open a pull request against `main`

## Commit conventions

Commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add X
fix: correct Y
refactor: simplify Z
test: add tests for W
ci: update workflow
docs: update README
chore: bump dependencies
```

Breaking changes: append `!` after the type, e.g. `feat!: rename flag`.

## PR checklist

- [ ] Tests pass (`go test ./...`)
- [ ] No new lint errors (`golangci-lint run`)
- [ ] `go mod tidy` was run if dependencies changed
- [ ] Relevant docs updated (README, llms.txt)

## Project structure

```
main.go               entry point
internal/
  agent/              agent detection heuristics
  socket/             Unix socket listener
  state/              thread-safe state store
  tmux/               tmux interface and live polling
  ui/                 Bubbletea TUI
agent-hook-bridge     Claude Code hook script
clank.tmux            TPM plugin entry point
scripts/install.sh    binary download script
```
