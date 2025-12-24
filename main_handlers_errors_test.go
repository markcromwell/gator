package main

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/markcromwell/gator/internal/database"
)

func TestHandlerAddFeed_InvalidArgs(t *testing.T) {
	s, _, cleanup := makeStateWithMock(t)
	defer cleanup()

	currentUser := database.User{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "bob"}
	if err := handlerAddFeed(s, command{name: "addfeed", arguments: []string{"only-name"}}, currentUser); err == nil {
		t.Fatalf("expected error for missing URL argument")
	}
}

func TestHandlerFollow_InvalidURL(t *testing.T) {
	s, _, cleanup := makeStateWithMock(t)
	defer cleanup()

	currentUser := database.User{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "bob"}
	if err := handlerFollow(s, command{name: "follow", arguments: []string{"not-a-url"}}, currentUser); err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}

func TestHandlerUnfollow_InvalidURL(t *testing.T) {
	s, _, cleanup := makeStateWithMock(t)
	defer cleanup()

	currentUser := database.User{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: "bob"}
	if err := handlerUnfollow(s, command{name: "unfollow", arguments: []string{"not-a-url"}}, currentUser); err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}

func TestHandlerLogin_UserNotFound(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	// Return no rows for GetUserByName
	mock.ExpectQuery(`(?i)SELECT .+ FROM users WHERE name`).WillReturnError(sql.ErrNoRows)

	err := handlerLogin(s, command{name: "login", arguments: []string{"noone"}})
	if err == nil {
		t.Fatalf("expected error when user not found")
	}
}

func TestHandlerRegister_DBError(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)INSERT INTO users`).WillReturnError(errors.New("db insert failed"))

	err := handlerRegister(s, command{name: "register", arguments: []string{"newuser"}})
	if err == nil {
		t.Fatalf("expected error when DB insert fails")
	}
}

func TestHandlerLogin_DBError(t *testing.T) {
	s, mock, cleanup := makeStateWithMock(t)
	defer cleanup()

	mock.ExpectQuery(`(?i)SELECT .+ FROM users WHERE name`).WillReturnError(fmt.Errorf("db error"))

	err := handlerLogin(s, command{name: "login", arguments: []string{"alice"}})
	if err == nil {
		t.Fatalf("expected error when DB query fails")
	}
}
