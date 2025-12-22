package main

import (
	"errors"
	"testing"
)

func TestCommandsRegisterAndRun(t *testing.T) {
	cmds := &commands{commandsMap: make(map[string]func(*state, command) error)}

	called := false
	if err := cmds.register("doit", func(s *state, c command) error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("register error: %v", err)
	}

	if err := cmds.run(&state{}, command{name: "doit"}); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !called {
		t.Fatalf("expected handler to be called")
	}

	// unknown command should return an error
	if err := cmds.run(&state{}, command{name: "nope"}); err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestCommandsHandlerErrorPropagation(t *testing.T) {
	cmds := &commands{commandsMap: make(map[string]func(*state, command) error)}
	if err := cmds.register("bad", func(s *state, c command) error {
		return errors.New("boom")
	}); err != nil {
		t.Fatalf("register error: %v", err)
	}
	if err := cmds.run(&state{}, command{name: "bad"}); err == nil || err.Error() != "boom" {
		t.Fatalf("expected propagated error 'boom', got: %v", err)
	}
}
