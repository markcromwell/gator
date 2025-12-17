package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/markcromwell/gator/internal/config"
	"github.com/markcromwell/gator/internal/database"
	"github.com/markcromwell/gator/internal/feed"
)

type state struct {
	config    *config.Config
	db        *sql.DB
	dbQueries *database.Queries
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

func (c *commands) register(name string, handler func(*state, command) error) error {
	c.commandsMap[name] = handler
	return nil
}

func handlerReset(s *state, cmd command) error {
	fmt.Println("Resetting users table...")
	err := s.dbQueries.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error resetting users table: %w", err)
	}
	fmt.Println("Users table reset successfully.")
	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("username argument is required")
	}

	username := cmd.arguments[0]
	ctx := context.Background()

	// Check if user exists
	_, err := s.dbQueries.GetUserByName(ctx, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user %s not found", username)
		}
		return fmt.Errorf("error checking user existence: %w", err)
	}

	fmt.Printf("Logging in as %s\n", username)
	return s.config.SetUser(username)
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.arguments) < 2 {
		return fmt.Errorf("feed name and URL arguments are required")
	}

	feedName := cmd.arguments[0]
	feedURL := cmd.arguments[1]

	currentUser, err := s.dbQueries.GetUserByName(context.Background(), s.config.CurrentUserName)
	if err != nil {
		return fmt.Errorf("get current user: %w", err)
	}

	fmt.Printf("Adding feed %s with URL %s for user %s\n", feedName, feedURL, currentUser.Name)

	newFeed, err := s.dbQueries.CreateFeeds(context.Background(), database.CreateFeedsParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      feedName,
		Url:       feedURL,
		UserID:    currentUser.ID,
	})
	if err != nil {
		return fmt.Errorf("create feed: %w", err)
	}

	fmt.Println("Feed added successfully.")
	fmt.Println("Feed ID:", newFeed.ID.String())
	fmt.Println("Feed Name:", newFeed.Name)
	fmt.Println("Feed URL:", newFeed.Url)
	fmt.Println("Associated User ID:", newFeed.UserID.String())
	fmt.Println("Created At:", newFeed.CreatedAt)
	fmt.Println("Updated At:", newFeed.UpdatedAt)
	fmt.Println("Process complete.")
	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) == 0 {
		return fmt.Errorf("username argument is required")
	}

	username := cmd.arguments[0]

	fmt.Printf("Registering user %s\n", username)

	user, error := s.dbQueries.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),       // Generates a new UUID v4.
		CreatedAt: time.Now().UTC(), // Use current UTC time for creation timestamp.
		UpdatedAt: time.Now().UTC(), // Same for update timestamp (often set to CreatedAt initially).
		Name:      username,         // Example name string.
	})

	if error != nil {
		return error
	}

	fmt.Printf("User registered with ID: %s\n", user.ID.String())
	return s.config.SetUser(username)
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("no arguments expected for users command")
	}

	users, err := s.dbQueries.GetUsers(context.Background(), database.GetUsersParams{Limit: 1000, Offset: 0})

	if err != nil {
		return err
	}

	for _, user := range users {
		current := ""
		if s.config.CurrentUserName == user.Name {
			current = " (current)"
		}
		fmt.Printf("* %s%s\n", user.Name, current)
	}

	return nil
}

func handlerAgg(s *state, cmd command) error {
	ctx := context.Background()
	feedURL := "https://www.wagslane.dev/index.xml"
	f, err := feed.FetchFeed(ctx, feedURL)
	if err != nil {
		return fmt.Errorf("fetch feed: %w", err)
	}
	fmt.Printf("%+v\n", f)
	return nil
}

func main() {
	conf, err := config.Read()
	if err != nil {
		fmt.Println("Error reading config:", err)
		os.Exit(1)
	}

	cmdState := &state{config: conf}

	db, err := sql.Open("postgres", conf.DbURL)
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		os.Exit(1)
	}
	defer db.Close()
	cmdState.db = db
	cmdState.dbQueries = database.New(db)

	cmds := &commands{commandsMap: make(map[string]func(*state, command) error)}
	if err := cmds.register("login", handlerLogin); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("register", handlerRegister); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("reset", handlerReset); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("users", handlerUsers); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("agg", handlerAgg); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("addfeed", handlerAddFeed); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}

	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: gator <command> [arguments]")
		os.Exit(1)
	}

	cmdName := args[1]
	cmdArgs := args[2:]
	cmd2Run := command{name: cmdName, arguments: cmdArgs}
	err = cmds.run(cmdState, cmd2Run)
	db.Close()
	if err != nil {
		fmt.Println("Error executing command:", err)
		os.Exit(1)
	}

	os.Exit(0)
}
