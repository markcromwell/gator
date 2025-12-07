<!-- .github/copilot-instructions.md -->
# Copilot / AI Agent Instructions for the `gator` repo

Purpose: give AI coding agents concise, actionable context so they can be productive
quickly in this small Go project.

- Repository root: module `github.com/markcromwell/gator` (see `go.mod`). Go version: `1.25.1`.
- Primary artifact: a single command-line binary built from `main.go`.

Big picture
- This is a small Go CLI app. `main.go` is the program entrypoint (currently prints "Hello, Gator!").
- Internal packages live under `internal/` (example: `internal/config`). Treat `internal/*` as
  non-public implementation details — do not move them to public import paths.

Key patterns & files
- `internal/config/config.go`:
  - Reads JSON config from `$HOME/.gatorconfig.json` using `os.UserHomeDir()` and `filepath.Join`.
  - Exposes `type Config struct { DbURL string `json:"db_url"` }` and `func Read() (*Config, error)`.
  - When modifying config structure keep JSON tags and the `$HOME/.gatorconfig.json` behaviour in mind.
- `main.go`:
  - The program entrypoint; minimal now. If you add startup logic, keep it small and call into `internal/*` packages.

Developer workflows (commands you can rely on)
- Build locally: `go build -o gator .`
- Run quickly: `go run .` or `./gator` after building.
- Formatting & vetting: `gofmt -w .` and `go vet ./...`.
- Tests: none currently — add package-level `_test.go` files next to code under test and run `go test ./...`.

Project-specific conventions
- Use `internal/` for non-exported app code; prefer small packages (config, handlers, etc.).
- Config file location is intentionally user-local: `$HOME/.gatorconfig.json` and uses snake_case JSON keys
  (example `db_url`). If you add flags or environment variable support, document precedence (flags > env > config file).
- Errors: functions return `(T, error)`; propagate errors upward rather than swallowing them.

Integration points & external dependencies
- No external services are wired yet in repo. `DbURL` in `Config` implies a future DB connection — add connection logic
  in a new package (e.g., `internal/db`) and keep connection initialization separate from main.

When an AI agent should modify code
- Preserve module path and `internal/` boundaries.
- Run `go build` after edits and prefer changing only necessary files.
- Add tests for behaviour you change. Keep tests in the same package and run `go test ./...` before proposing PRs.

Examples (how to read config in `main.go`)
```go
cfg, err := config.Read()
if err != nil {
    // return or log and exit
}
fmt.Println(cfg.DbURL)
```

PR & commit guidance for AI agents
- Make focused commits (one logical change per commit).
- Update or add a brief note in `README.md` if you add user-visible behaviour (CLI flags, config keys, etc.).

What not to assume
- There are no tests or CI files to infer additional requirements — don't invent CI rules.
- The repo is intentionally small; avoid large architectural rewrites without human approval.

If anything is unclear, ask the human owner for clarification: common follow-ups include desired
config precedence (flags/env/file), expected DB driver, and whether cross-platform config locations
should be supported.

— End
