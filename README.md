# Gator

> A tiny CLI feed reader and scraper written in Go.

Prerequisites
- Go (tested with Go 1.25)
- PostgreSQL (local or remote) with a database and user for the app

Install

1. Build/install the CLI (from the repository root):

```bash
# installs to $(go env GOPATH)/bin or $GOBIN
go install github.com/markcromwell/gator@latest
```

Configuration

Gator reads its DB URL and current user from a JSON config file in your home directory: `~/.gatorconfig.json`.
Create `~/.gatorconfig.json` with contents like:

```json
{
  "db_url": "postgres://gator:pass@127.0.0.1:5432/gator?sslmode=disable",
  "current_user_name": "yourusername"
}
```

Make sure the `db_url` user has appropriate privileges and the DB contains the schema (SQL files are in `sql/schema`).

Running

From the repo root you can run the CLI directly during development:

```bash
# run a single command
go run . <command> [arguments]

# examples:
go run . register alice
go run . login alice
go run . addfeed "xkcd" https://xkcd.com/rss.xml
go run . following
go run . browse 10
```

The scraper runs a loop and stores posts in the DB:

```bash
# scrape every second (for testing)
go run . scrapeFeeds 1s

# aggregator (prints items but doesn't save posts)
go run . agg 1s
```

Notes
- The `scrapeFeeds` command selects feeds whose `last_fetched_at` is NULL or older than 10 minutes.
- If you change SQL in `sql/queries` run `sqlc generate` to regenerate the typed queries.
- Tests: `go test ./...` runs unit and integration tests (integration tests require `GATOR_TEST_DB` or a working DB configured in `~/.gatorconfig.json`).

Contributing
- Create a branch, make focused commits, and open a PR. Keep changes small and include tests for behavior you change.

License
- No license file is included in this repository.
