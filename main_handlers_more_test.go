package main

import (
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/markcromwell/gator/internal/database"
)

func TestHandlerFeeds(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	fid := uuid.New()
	uid := uuid.New()
	now := time.Now()

	// feed rows: id, created_at, last_fetched_at, updated_at, name, url, user_id
	feedRows := sqlmock.NewRows([]string{"id", "created_at", "last_fetched_at", "updated_at", "name", "url", "user_id"}).
		AddRow(fid, now, nil, now, "feed1", "https://example.com/feed", uid)

	mock.ExpectQuery(`(?i)SELECT .+ FROM feeds ORDER BY`).WillReturnRows(feedRows)

	// expect user lookup
	userRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name"}).
		AddRow(uid, now, now, "bob")
	mock.ExpectQuery(`SELECT .+ FROM users WHERE id`).WillReturnRows(userRows)

	out := captureStdout(t, func() {
		if err := handlerFeeds(s, command{name: "feeds"}); err != nil {
			t.Fatalf("handlerFeeds: %v", err)
		}
	})

	if !strings.Contains(out, "feed1") || !strings.Contains(out, "https://example.com/feed") || !strings.Contains(out, "bob") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestHandlerFollowing(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	uid := uuid.New()
	fid := uuid.New()
	ffid := uuid.New()
	now := time.Now()

	// GetFeedFollowsByUserID columns: id, created_at, updated_at, user_id, feed_id, user_name, feed_name
	followsRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at", "user_id", "feed_id", "user_name", "feed_name"}).
		AddRow(ffid, now, now, uid, fid, "bob", "f1")
	mock.ExpectQuery(`(?i)SELECT .+ FROM feed_follows`).WillReturnRows(followsRows)

	// GetFeedByID returns feed row
	feedRows := sqlmock.NewRows([]string{"id", "created_at", "last_fetched_at", "updated_at", "name", "url", "user_id"}).
		AddRow(fid, now, nil, now, "f1", "https://example.com", uid)
	mock.ExpectQuery(`(?i)SELECT .+ FROM feeds WHERE id`).WillReturnRows(feedRows)

	currentUser := database.User{ID: uid, CreatedAt: now, UpdatedAt: now, Name: "bob"}

	out := captureStdout(t, func() {
		if err := handlerFollowing(s, command{name: "following"}, currentUser); err != nil {
			t.Fatalf("handlerFollowing: %v", err)
		}
	})

	if !strings.Contains(out, "f1") || !strings.Contains(out, "https://example.com") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestHandlerUnfollow(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	uid := uuid.New()
	fid := uuid.New()
	now := time.Now()

	// GetFeedByURL
	feedRows := sqlmock.NewRows([]string{"id", "created_at", "last_fetched_at", "updated_at", "name", "url", "user_id"}).
		AddRow(fid, now, nil, now, "f1", "https://example.com", uid)
	mock.ExpectQuery(`(?i)SELECT .+ FROM feeds WHERE url`).WillReturnRows(feedRows)

	// Expect delete exec
	mock.ExpectExec(`DELETE FROM feed_follows`).WillReturnResult(sqlmock.NewResult(0, 1))

	currentUser := database.User{ID: uid, CreatedAt: now, UpdatedAt: now, Name: "bob"}

	if err := handlerUnfollow(s, command{name: "unfollow", arguments: []string{"https://example.com"}}, currentUser); err != nil {
		t.Fatalf("handlerUnfollow: %v", err)
	}
}

func TestMiddlewareLoggedIn(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	// when not logged in
	s.config.CurrentUserName = ""
	wrapped := middlewareLoggedIn(func(s *state, cmd command, user database.User) error { return nil })
	if err := wrapped(s, command{name: "x"}); err == nil {
		t.Fatalf("expected error when not logged in")
	}

	// when logged in - expect GetUserByName
	s.config.CurrentUserName = "alice"
	id := uuid.New()
	now := time.Now()
	mock.ExpectQuery(`SELECT .+ FROM users WHERE name`).WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at", "name"}).AddRow(id, now, now, "alice"))

	called := false
	wrapped2 := middlewareLoggedIn(func(s *state, cmd command, user database.User) error {
		called = true
		if user.Name != "alice" {
			t.Fatalf("expected alice, got %s", user.Name)
		}
		return nil
	})

	if err := wrapped2(s, command{name: "y"}); err != nil {
		t.Fatalf("middlewareLoggedIn wrapped handler returned error: %v", err)
	}
	if !called {
		t.Fatalf("wrapped handler was not called")
	}
}
