package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/markcromwell/gator/internal/config"
	"github.com/markcromwell/gator/internal/database"
)

// makeStateWithMock creates a test state with a sqlmock database.
// It sets HOME to a temp dir so config writes are isolated.
func makeStateWithMock(t *testing.T) (*state, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	qs := database.New(db)

	// Isolated HOME for config file writes
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cfg := &config.Config{DbURL: "postgres://db"}
	s := &state{config: cfg, db: db, dbQueries: qs}

	cleanup := func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet sqlmock expectations: %v", err)
		}
		db.Close()
	}
	return s, mock, cleanup
}

// captureStdout captures stdout during fn execution and returns the output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan string)
	go func() {
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				b.Write(buf[:n])
			}
			if readErr != nil {
				break
			}
		}
		done <- b.String()
	}()

	fn()
	w.Close()
	os.Stdout = orig
	return <-done
}

func TestHandlerRegister(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	// Match the INSERT INTO users query
	mock.ExpectQuery(`INSERT INTO users`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name"}).
			AddRow(id, now, now, "newuser"))

	err := handlerRegister(s, command{name: "register", arguments: []string{"newuser"}})
	if err != nil {
		t.Fatalf("handlerRegister: %v", err)
	}

	// Verify config was updated
	cfg, err := config.Read()
	if err != nil {
		t.Fatalf("config.Read: %v", err)
	}
	if cfg.CurrentUserName != "newuser" {
		t.Errorf("expected CurrentUserName=newuser, got %q", cfg.CurrentUserName)
	}
}

func TestHandlerReset(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	// Match DELETE FROM users
	mock.ExpectExec(`DELETE FROM users`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := handlerReset(s, command{name: "reset"})
	if err != nil {
		t.Fatalf("handlerReset: %v", err)
	}
}

func TestHandlerLogin(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	id := uuid.New()
	now := time.Now()

	// Match SELECT ... FROM users WHERE name = $1
	mock.ExpectQuery(`SELECT .+ FROM users WHERE name`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name"}).
			AddRow(id, now, now, "alice"))

	err := handlerLogin(s, command{name: "login", arguments: []string{"alice"}})
	if err != nil {
		t.Fatalf("handlerLogin: %v", err)
	}

	cfg, err := config.Read()
	if err != nil {
		t.Fatalf("config.Read: %v", err)
	}
	if cfg.CurrentUserName != "alice" {
		t.Errorf("expected CurrentUserName=alice, got %q", cfg.CurrentUserName)
	}
}

func TestHandlerUsers(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	id1 := uuid.New()
	id2 := uuid.New()
	now := time.Now()

	rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name"}).
		AddRow(id1, now, now, "u1").
		AddRow(id2, now, now, "u2")

	// Match SELECT ... FROM users ORDER BY
	mock.ExpectQuery(`SELECT .+ FROM users ORDER BY`).
		WillReturnRows(rows)

	out := captureStdout(t, func() {
		if err := handlerUsers(s, command{name: "users"}); err != nil {
			t.Fatalf("handlerUsers: %v", err)
		}
	})

	if !strings.Contains(out, "u1") || !strings.Contains(out, "u2") {
		t.Errorf("expected output to contain u1 and u2, got: %s", out)
	}
}

func TestHandlerAddFeed(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	uid := uuid.New()
	fid := uuid.New()
	ffid := uuid.New()
	now := time.Now()

	// CreateFeeds INSERT
	mock.ExpectQuery(`INSERT INTO feeds`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "url", "user_id"}).
			AddRow(fid, now, now, "f1", "https://example.com/feed", uid))

	// CreateFeedFollow (WITH inserted AS ...)
	mock.ExpectQuery(`WITH inserted AS`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "feed_id", "user_name", "feed_name"}).
			AddRow(ffid, now, now, uid, fid, "bob", "f1"))

	currentUser := database.User{ID: uid, CreatedAt: now, UpdatedAt: now, Name: "bob"}

	out := captureStdout(t, func() {
		err := handlerAddFeed(s, command{
			name:      "addfeed",
			arguments: []string{"f1", "https://example.com/feed"},
		}, currentUser)
		if err != nil {
			t.Fatalf("handlerAddFeed: %v", err)
		}
	})

	if !strings.Contains(out, "Feed added successfully") {
		t.Errorf("expected 'Feed added successfully' in output, got: %s", out)
	}
}

func TestHandlerFollow(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	uid := uuid.New()
	fid := uuid.New()
	ffid := uuid.New()
	now := time.Now()

	// GetFeedByURL
	mock.ExpectQuery(`SELECT .+ FROM feeds`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name", "url", "user_id"}).
			AddRow(fid, now, now, "feed1", "https://example.com/feed", uid))

	// CreateFeedFollow (WITH inserted AS ...)
	mock.ExpectQuery(`WITH inserted AS`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "feed_id", "user_name", "feed_name"}).
			AddRow(ffid, now, now, uid, fid, "bob", "feed1"))

	currentUser := database.User{ID: uid, CreatedAt: now, UpdatedAt: now, Name: "bob"}

	out := captureStdout(t, func() {
		err := handlerFollow(s, command{
			name:      "follow",
			arguments: []string{"https://example.com/feed"},
		}, currentUser)
		if err != nil {
			t.Fatalf("handlerFollow: %v", err)
		}
	})

	if !strings.Contains(out, "Feed follow created successfully") {
		t.Errorf("expected 'Feed follow created successfully' in output, got: %s", out)
	}
}
