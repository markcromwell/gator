package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestIntegrationCommands runs a small integration flow against a real Postgres
// database. It is skipped unless the GATOR_TEST_DB environment variable is set.
// The test will apply schema files from the sql/schema directory and then run
// several CLI commands (register, login, addfeed, feeds) using `go run .`.
func TestIntegrationCommands(t *testing.T) {
	dbURL := os.Getenv("GATOR_TEST_DB")
	if dbURL == "" {
		t.Skip("skipping integration test; set GATOR_TEST_DB to run")
	}

	// connect and apply schema
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	// reset database to clean state
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatalf("reset db schema: %v", err)
	}

	// run schema files
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	schemaDir := filepath.Join(root, "sql", "schema")
	files, err := ioutil.ReadDir(schemaDir)
	if err != nil {
		t.Fatalf("read schema dir: %v", err)
	}
	// ensure schema files are applied in name order (001_..., 002_...)
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		raw, err := ioutil.ReadFile(filepath.Join(schemaDir, f.Name()))
		if err != nil {
			t.Fatalf("read schema file: %v", err)
		}
		content := string(raw)
		// if file contains goose markers, extract the Up block only
		if strings.Contains(content, "-- +goose Up") {
			upIdx := strings.Index(content, "-- +goose Up")
			downIdx := strings.Index(content, "-- +goose Down")
			if upIdx >= 0 && downIdx > upIdx {
				// extract between the markers
				content = content[upIdx+len("-- +goose Up") : downIdx]
			}
		}
		if _, err := db.Exec(content); err != nil {
			t.Fatalf("apply %s: %v", f.Name(), err)
		}
		// quick sanity check: if we just applied users migration, ensure table exists
		if strings.Contains(f.Name(), "001_users") {
			var tbl sql.NullString
			err := db.QueryRow("SELECT to_regclass('public.users')").Scan(&tbl)
			if err != nil {
				t.Fatalf("verify users table query failed: %v", err)
			}
			if !tbl.Valid {
				t.Fatalf("users table not found after applying %s", f.Name())
			}
		}
	}

	// Prepare a clean HOME with config file pointing to our DB
	home := t.TempDir()
	cfg := map[string]string{"db_url": dbURL}
	cfgBytes, _ := json.Marshal(cfg)
	if err := ioutil.WriteFile(filepath.Join(home, ".gatorconfig.json"), cfgBytes, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// build the binary once in the temp HOME to avoid go tool creating files
	binPath := filepath.Join(home, "gator-cli")
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	// build using the normal HOME so the go tool's cache isn't created inside our tempdir
	buildCmd.Env = os.Environ()
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("go build failed: %v\nout: %s", err, string(out))
	}

	// helper to run the built CLI with the temporary HOME
	run := func(ctx context.Context, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, binPath, args...)
		cmd.Env = append(os.Environ(), "HOME="+home)
		out, err := cmd.CombinedOutput()
		return string(out), err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// register user
	out, err := run(ctx, "register", "itestuser")
	if err != nil {
		t.Fatalf("register failed: %v\nout: %s", err, out)
	}
	if !contains(out, "User registered") {
		t.Fatalf("unexpected register output: %s", out)
	}

	// login
	out, err = run(ctx, "login", "itestuser")
	if err != nil {
		t.Fatalf("login failed: %v\nout: %s", err, out)
	}

	// add a feed
	feedURL := "https://example.com/feed"
	out, err = run(ctx, "addfeed", "IT Test Feed", feedURL)
	if err != nil {
		t.Fatalf("addfeed failed: %v\nout: %s", err, out)
	}

	// list feeds
	out, err = run(ctx, "feeds")
	if err != nil {
		t.Fatalf("feeds failed: %v\nout: %s", err, out)
	}
	if !contains(out, "IT Test Feed") || !contains(out, feedURL) {
		t.Fatalf("unexpected feeds output: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (s != "" && sub != "" && (stringIndex(s, sub) >= 0)))
}

// minimal stringIndex to avoid importing strings package repeatedly
func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
