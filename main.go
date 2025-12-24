package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("no arguments expected for feeds command")
	}

	feeds, err := s.dbQueries.GetFeed(context.Background(), database.GetFeedParams{Limit: 1000, Offset: 0})
	if err != nil {
		return fmt.Errorf("error fetching feeds: %w", err)
	}

	for _, feed := range feeds {
		user, err := s.dbQueries.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("error fetching user for feed %s: %w", feed.ID.String(), err)
		}

		fmt.Printf("* %s (%s) - %s - %s\n", feed.Name, feed.ID.String(), feed.Url, user.Name)
	}

	return nil
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		if s.config.CurrentUserName == "" {
			return fmt.Errorf("no user is currently logged in")
		}

		user, err := s.dbQueries.GetUserByName(context.Background(), s.config.CurrentUserName)
		if err != nil {
			return fmt.Errorf("error fetching logged-in user: %w", err)
		}

		return handler(s, cmd, user)
	}
}
func handlerAddFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 2 {
		return fmt.Errorf("feed name and URL arguments are required")
	}

	feedName := cmd.arguments[0]
	feedURL := cmd.arguments[1]

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

	// add the feed to the user's follows
	_, err = s.dbQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    currentUser.ID,
		FeedID:    newFeed.ID,
	})
	if err != nil {
		return fmt.Errorf("create feed follow: %w", err)
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
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("time_between_reqs argument is required")
	}

	intervalStr := cmd.arguments[0]
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	fmt.Printf("Collecting feeds every %s\n", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigc)

	ctx := context.Background()

	// Run immediately, then on each tick. Outer loop waits for ticker or signal.
	for {
		// Try up to 10 feeds per tick
		for i := 0; i < 10; i++ {
			f, err := s.dbQueries.GetNextFeedToFetch(ctx)
			if err != nil {
				// no ready feeds or DB error; stop inner loop and wait for next tick
				if err == sql.ErrNoRows {
					break
				}
				fmt.Println("Error selecting next feed to fetch:", err)
				break
			}

			fmt.Println("Fetching feed:", f.Url)
			feedData, err := feed.FetchFeed(ctx, f.Url)
			if err != nil {
				fmt.Println("Error fetching feed:", err)
			} else {
				for _, item := range feedData.Channel.Item {
					fmt.Printf("- %s\n  %s\n", item.Title, item.Link)
				}
			}

			if err := s.dbQueries.MarkFeedFetched(ctx, f.ID); err != nil {
				fmt.Println("Error marking feed fetched:", err)
			}
			// be polite to remote servers — wait a bit between requests
			time.Sleep(1 * time.Second)
		}

		select {
		case <-ticker.C:
			continue
		case <-sigc:
			fmt.Println("received interrupt; exiting agg")
			return nil
		}
	}
}

/*
	handlerFollows - takes a single url argument and creates a new feed follow record

-- for the current user. It should print the name of the feed and the
-- current user once the record is created
*/
func handlerFollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 1 {
		return fmt.Errorf("feed URL argument is required")
	}

	feedURL := cmd.arguments[0]

	// Verify feedURL is a valid URL
	_, err := url.ParseRequestURI(feedURL)
	if err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}

	feedRecord, err := s.dbQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("get feed by URL: %w", err)
	}

	newFollow, err := s.dbQueries.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		UserID:    currentUser.ID,
		FeedID:    feedRecord.ID,
	})
	if err != nil {
		return fmt.Errorf("create feed follow: %w", err)
	}

	fmt.Println("Feed follow created successfully.")
	fmt.Println("Feed Name:", feedRecord.Name)
	fmt.Println("Followed by User:", currentUser.Name)
	fmt.Println("Follow ID:", newFollow.ID.String())

	return nil
}

/*
following shows a list of all feed follows for the current user including feed name and URL
*/
func handlerFollowing(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("no arguments expected for following command")
	}

	follows, err := s.dbQueries.GetFeedFollowsByUserID(context.Background(), database.GetFeedFollowsByUserIDParams{UserID: currentUser.ID, Limit: 1000, Offset: 0})
	if err != nil {
		return fmt.Errorf("get feed follows: %w", err)
	}

	for _, follow := range follows {
		feedRecord, err := s.dbQueries.GetFeedByID(context.Background(), follow.FeedID)
		if err != nil {
			return fmt.Errorf("get feed by ID: %w", err)
		}
		fmt.Printf("* %s - %s\n", feedRecord.Name, feedRecord.Url)
	}

	return nil
}

// handlerUnfollow removes a feed follow for the current user based on feed ID argument
func handlerUnfollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) < 1 {
		return fmt.Errorf("feed ID argument is required")
	}
	// The argument must be a feed URL (the feed identifier). Do not accept raw UUIDs.
	feedURL := cmd.arguments[0]

	if _, err := url.ParseRequestURI(feedURL); err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}

	// Lookup feed by URL to get the feed ID; error if not found
	f, err := s.dbQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("get feed by URL: %w", err)
	}

	err = s.dbQueries.DeleteFeedFollowByUserIDAndFeedID(context.Background(), database.DeleteFeedFollowByUserIDAndFeedIDParams{
		FeedID: f.ID,
		UserID: currentUser.ID,
	})
	if err != nil {
		return fmt.Errorf("delete feed follow: %w", err)
	}

	fmt.Println("Feed unfollowed successfully.")
	return nil
}

// convert from string to sql.NullString
func strToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

// ParseFeedDate attempts to parse a date string from RSS/Atom feeds using a comprehensive list of common formats.
// It supports variations from RFC 822 (RSS) and RFC 3339/ISO 8601 (Atom), including:
// - With/without weekdays
// - Days with/without leading zeros
// - Named timezones (e.g., MST, EST)
// - Numeric offsets (e.g., -0700, +0200)
// - UTC 'Z' designator
// - Fractional seconds
// - Rare cases with space instead of 'T' in ISO formats
//
// If parsing fails for all formats, it returns a zero time.Time and an error.
func ParseFeedDate(dateStr string) (time.Time, error) {
	formats := []string{
		// RFC 822 / RSS variations (with weekday)
		time.RFC1123,                   // "Mon, 02 Jan 2006 15:04:05 MST" (named TZ)
		time.RFC1123Z,                  // "Mon, 02 Jan 2006 15:04:05 -0700" (numeric TZ)
		"Mon, 2 Jan 2006 15:04:05 MST", // Day without leading zero, named TZ
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 +0000", // +0000 offset
		"Mon, 2 Jan 2006 15:04:05 +0000",

		// Without weekday
		"02 Jan 2006 15:04:05 MST",
		"02 Jan 2006 15:04:05 -0700",
		"2 Jan 2006 15:04:05 MST",
		"2 Jan 2006 15:04:05 -0700",
		"02 Jan 2006 15:04:05 +0000",
		"2 Jan 2006 15:04:05 +0000",

		// RFC 3339 / Atom variations
		time.RFC3339,                     // "2006-01-02T15:04:05Z07:00" (wait, actually "2006-01-02T15:04:05-07:00")
		time.RFC3339Nano,                 // With nanoseconds: "2006-01-02T15:04:05.999999999-07:00"
		"2006-01-02T15:04:05Z",           // Z without offset
		"2006-01-02T15:04:05.999999999Z", // Z with nano
		"2006-01-02T15:04:05+00:00",      // +00:00
		"2006-01-02T15:04:05-00:00",      // -00:00

		// Rare: Space instead of 'T' (some non-standard feeds)
		"2006-01-02 15:04:05Z",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05+00:00",
		"2006-01-02 15:04:05.999999999Z",

		// Other observed formats from feeds
		"Mon, 02 Jan 2006 15:04:05 GMT", // GMT specifically
		"Mon, 2 Jan 2006 15:04:05 GMT",
		"02 Jan 2006 15:04:05 GMT",
		"2 Jan 2006 15:04:05 GMT",
		"Mon, 02 Jan 2006 15:04:05 EST", // EST, etc.
	}

	var parsedTime time.Time
	var lastErr error
	for _, format := range formats {
		parsedTime, lastErr = time.Parse(format, dateStr)
		if lastErr == nil {
			return parsedTime.UTC(), nil // Normalize to UTC for consistency
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date '%s' with any known format: %w", dateStr, lastErr)
}

// handlerScrapeFeeds - runs in the background to scrape all feeds and store new items.
func handlerScrapeFeeds(s *state, cmd command) error {
	// takes 1 parameter: interval in seconds, minutes or hours, or days 1s etc.
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("interval argument is required")
	}
	intervalStr := cmd.arguments[0]
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return fmt.Errorf("invalid interval: %w", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fmt.Printf("Starting feed scraping every %s %s\n", interval, s.config.CurrentUserName)

	ctx := context.Background()

	for {
		<-ticker.C
		fmt.Println("Scraping feeds...")

		// Fetch up to 10 feeds to process this tick using GetNextFeedToFetch
		for i := 0; i < 10; i++ {
			f, err := s.dbQueries.GetNextFeedToFetch(ctx)
			if err != nil {
				// no feed ready or other error
				if err == sql.ErrNoRows {
					break
				}
				fmt.Println("Error selecting next feed to fetch:", err)
				break
			}

			fmt.Println("Fetching feed:", f.Url)
			feedData, err := feed.FetchFeed(ctx, f.Url)
			if err != nil {
				fmt.Println("Error fetching feed:", err)
			} else {
				for _, item := range feedData.Channel.Item {
					postDate, postErr := ParseFeedDate(item.PubDate)
					if postErr != nil {
						fmt.Println("Error parsing post date:", postErr)
						continue
					}

					fmt.Printf("- %s\n  %s\n", item.Title, item.Link)
					// Insert the post into the database
					_, err := s.dbQueries.CreatePost(ctx, database.CreatePostParams{
						ID:          uuid.New(),
						CreatedAt:   time.Now().UTC(),
						UpdatedAt:   time.Now().UTC(),
						Title:       item.Title,
						Url:         item.Link,
						Description: strToNullString(item.Description),
						PublishedAt: postDate,
						FeedID:      f.ID,
					})
					if err != nil {
						fmt.Println("Error inserting post into database:", err)
					}
				}
			}

			if err := s.dbQueries.MarkFeedFetched(ctx, f.ID); err != nil {
				fmt.Println("Error marking feed fetched:", err)
			}

			// be polite to remote servers
			time.Sleep(1 * time.Second)
		}
	}

}

// handlerBrowse command. It should take an optional "limit" parameter. If it's not provided, default the limit to 2. Print the posts in the terminal.
func handlerBrowse(s *state, cmd command, currentUser database.User) error {
	limit := 2
	if len(cmd.arguments) > 0 {
		var err error
		limit, err = strconv.Atoi(cmd.arguments[0])
		if err != nil {
			return fmt.Errorf("invalid limit argument: %w", err)
		}
	}

	posts, err := s.dbQueries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: currentUser.ID,
		Limit:  int32(limit),
		Offset: 0,
	})
	if err != nil {
		return fmt.Errorf("error fetching posts: %w", err)
	}

	for _, post := range posts {
		fmt.Printf("* %s\n  %s\n  Published at: %s\n", post.Title, post.Url, post.PublishedAt.String())
	}

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
	if err := cmds.register("addfeed", middlewareLoggedIn(handlerAddFeed)); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("feeds", handlerFeeds); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("follow", middlewareLoggedIn(handlerFollow)); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("following", middlewareLoggedIn(handlerFollowing)); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("unfollow", middlewareLoggedIn(handlerUnfollow)); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("scrapeFeeds", handlerScrapeFeeds); err != nil {
		fmt.Println("Error registering command:", err)
		os.Exit(1)
	}
	if err := cmds.register("browse", middlewareLoggedIn(handlerBrowse)); err != nil {
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
