package main

import (
	"context"
	"database/sql"
	"fmt"
	"gator/internal/config"
	"gator/internal/database"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	DBQueries *database.Queries
	cfg *config.Config
	userId uuid.UUID
}

type command struct {
	cmd string
	args []string
}

type commands struct {
	handlers map[string]func(*state, command) error
}

func (c *commands) run(st *state, cmd command) error {
	handler, exists := c.handlers[cmd.cmd]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.cmd)
	}

	return handler(st, cmd)
}

func (c *commands) register(name string, handler func(*state, command) error) {
	c.handlers[name] = handler
}

func loginHandler(st *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username is required for login")
	}

	user, err := st.DBQueries.GetUser(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("there was an error logging in: %w", err)
	}

	st.cfg.SetUser(user.Name)
	st.userId = user.ID
	fmt.Printf("Logged in as: %s\n", st.cfg.CurrentUserName)

	return nil
}

func registerHandler(st *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("username is required for registration")
	}

	user, err := st.DBQueries.CreateUser(context.Background(),
					database.CreateUserParams{
						ID: uuid.New(),
						CreatedAt: time.Now(),
						UpdatedAt: time.Now(),
						Name: cmd.args[0],
					})

	if err != nil {
		return fmt.Errorf("there was an error creating user: %w", err)
	}

	st.cfg.SetUser(user.Name)
	fmt.Printf("Created a new user: %s (%v)", user.Name, user.ID)

	return nil
}

func resetHandler(st *state, cmd command) error {
	if err := st.DBQueries.DeleteUsers(context.Background()); err != nil {
		return fmt.Errorf("there was error resetting the DB: %w", err)
	}

	return nil
}

func usersHandler(st *state, cmd command) error {
	users, err := st.DBQueries.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("there was an error trying to fetch users: %w", err)
	}

	for _, user := range users {
		if st.cfg.CurrentUserName == user {
			fmt.Printf("* %s (current)\n", user)
		} else {
			fmt.Printf("* %s\n", user)
		}
	}

	return nil
}

func aggHandler(st *state, cmd command) error {
	feedURL := "https://www.wagslane.dev/index.xml"

	_, err := FetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("there was a problem fetching/parsing the feed: %w", err)
	}

	return nil
}

func addFeedHandler(st *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("name and url is required to add a feed")
	}

	err := loginHandler(st, command{
		cmd: "login",
		args: []string{st.cfg.CurrentUserName},
	})

	// Check if user is logged in
    if st.userId == uuid.Nil {
        return fmt.Errorf("you must be logged in to add a feed")
    }

	feed, err := st.DBQueries.CreateFeed(context.Background(),
					database.CreateFeedParams{
						ID        :uuid.New(),
						CreatedAt :time.Now(),
						UpdatedAt :time.Now(),
						Name      :cmd.args[0],
						Url       :cmd.args[1],
						UserID    :st.userId,
					})
	if err != nil {
		return fmt.Errorf("there was an error creating feed: %w", err)
	}

	fmt.Printf("Created a new feed: %s (%v) [%s]", feed.Name, feed.ID, feed.Url)	

	return nil
}

func feedsHandler(st *state, cmd command) error {
	feeds, err := st.DBQueries.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("there was an error returning all feeds: %w", err)
	}

	for _, feed := range feeds {
		username, err := st.DBQueries.GetUserByID(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("problem fetching username. %w", err)
		}
		fmt.Printf("%s (%s)\n%s\n", feed.Name, feed.Url, username)
	}

	return nil
}

func main() {
	st := &state{}
	st.cfg = config.Read()

	db, err := sql.Open("postgres", st.cfg.DB_URL)
	if err != nil {
		fmt.Printf("Error connecting to database: %v\n", err)
		os.Exit(1)
	}
	st.DBQueries = database.New(db)

	cmds := &commands{handlers: make(map[string]func(*state, command) error)}
	cmds.register("login", loginHandler)
	cmds.register("register", registerHandler)
	cmds.register("users", usersHandler)
	cmds.register("reset", resetHandler)
	cmds.register("agg", aggHandler)
	cmds.register("addfeed", addFeedHandler)
	cmds.register("feeds", feedsHandler)

	if len(os.Args) < 2 {
		fmt.Println("No command provided.")
		os.Exit(1)
	}

	cmd := command{cmd: os.Args[1], args: os.Args[2:]}
	if err := cmds.run(st, cmd); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
