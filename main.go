package main

import (
	"fmt"
	"os"

	"github.com/markcromwell/gator/internal/config"
)

type state struct {
	config *config.Config
}

type command struct {
	name      string
	arguments []string
}

type commands struct {
	commandsMap map[string]func(*state, command) error
}

func (c *commands) run(s *state, cmd command) error {
	if handler, ok := c.commandsMap[cmd.name]; ok {
		return handler(s, cmd)
	}
	return fmt.Errorf("unknown command: %s", cmd.name)
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandsMap[name] = f
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("username argument is required")
	}

	username := cmd.arguments[0]
	fmt.Printf("Logging in as %s\n", username)
	return s.config.SetUser(username)
}

func main() {

	conf, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	cmdState := &state{config: conf}

	cmds := &commands{commandsMap: make(map[string]func(*state, command) error)}
	cmds.register("login", handlerLogin)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: gator <command> [arguments]")
		os.Exit(1)
	}

	cmdName := args[1]
	cmdArgs := args[2:]
	cmd2Run := command{name: cmdName, arguments: cmdArgs}
	if err := cmds.run(cmdState, cmd2Run); err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)
	}

	os.Exit(0)
}
