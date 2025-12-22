package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/markcromwell/gator/internal/config"
)

func TestReadAndSetUser(t *testing.T) {
	dir := t.TempDir()
	// point HOME to our temp dir
	t.Setenv("HOME", dir)

	cfgPath := filepath.Join(dir, ".gatorconfig.json")
	// create initial file
	initial := config.Config{DbURL: "postgres://example", CurrentUserName: "bob"}
	f, err := os.Create(cfgPath)
	if err != nil {
		t.Fatalf("create config file: %v", err)
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(initial); err != nil {
		t.Fatalf("write initial config: %v", err)
	}
	f.Close()

	// Read should return the values we wrote
	got, err := config.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if got.DbURL != initial.DbURL || got.CurrentUserName != initial.CurrentUserName {
		t.Fatalf("unexpected config: %#v", got)
	}

	// Test SetUser writes to disk
	c := &config.Config{DbURL: "x"}
	if err := c.SetUser("alice"); err != nil {
		t.Fatalf("SetUser error: %v", err)
	}
	// Read back
	got2, err := config.Read()
	if err != nil {
		t.Fatalf("Read after SetUser error: %v", err)
	}
	if got2.CurrentUserName != "alice" {
		t.Fatalf("expected current user alice, got %q", got2.CurrentUserName)
	}
}
