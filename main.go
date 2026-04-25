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
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type state struct {
	DBQueries *database.Queries
	cfg       *config.Config
	userId    uuid.UUID
}

type command struct {
	cmd  string
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
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
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
	if len(cmd.args) < 1 {
		return fmt.Errorf("please provide duration to aggregate over (e.g. 1h, 30m, etc.)")
	}

	timeBetweenReq, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	fmt.Println("Collecting feeds every", timeBetweenReq)

	feedURL := "https://www.wagslane.dev/index.xml"

	_, err = FetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("there was a problem fetching/parsing the feed: %w", err)
	}

	ticker := time.NewTicker(timeBetweenReq)

	for ; ; <-ticker.C {
		scrapeFeeds(st)
	}

	// return nil
}

func addFeedHandler(st *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("name and url is required to add a feed")
	}

	err := loginHandler(st, command{
		cmd:  "login",
		args: []string{st.cfg.CurrentUserName},
	})

	// Check if user is logged in
	if st.userId == uuid.Nil {
		return fmt.Errorf("you must be logged in to add a feed")
	}

	feed, err := st.DBQueries.CreateFeed(context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
			Url:       cmd.args[1],
			UserID:    st.userId,
		})
	if err != nil {
		return fmt.Errorf("there was an error creating feed: %w", err)
	}

	fmt.Printf("Created a new feed: %s (%v) [%s]", feed.Name, feed.ID, feed.Url)

	followHandler(st, command{
		cmd:  "follow",
		args: []string{feed.Url},
	})

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

func followHandler(st *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("feed URL is required to follow a feed")
	}

	err := loginHandler(st, command{
		cmd:  "login",
		args: []string{st.cfg.CurrentUserName},
	})

	// Check if user is logged in
	if st.userId == uuid.Nil {
		return fmt.Errorf("you must be logged in to follow a feed")
	}

	feedURL := cmd.args[0]

	// Fetch the RSS feed from the internet to get metadata
	rssFeed, err := FetchFeed(context.Background(), feedURL)
	if err != nil {
		return fmt.Errorf("there was an error fetching feed from URL: %w", err)
	}

	// Check if feed already exists in DB by URL
	dbFeed, err := st.DBQueries.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		// Feed doesn't exist, create it
		dbFeed, err = st.DBQueries.CreateFeed(context.Background(),
			database.CreateFeedParams{
				ID:        uuid.New(),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
				Name:      rssFeed.Channel.Title,
				Url:       feedURL,
				UserID:    st.userId,
			})
		if err != nil {
			return fmt.Errorf("there was an error creating feed in DB: %w", err)
		}
	}

	// Now create the feed follow relationship
	res, err := st.DBQueries.CreateFeedFollow(context.Background(),
		database.CreateFeedFollowParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			UserID:    st.userId,
			FeedID:    dbFeed.ID,
		})
	if err != nil {
		return fmt.Errorf("there was an error creating follow: %w", err)
	}

	fmt.Printf("%s is now following %s (%s)\n", res.UserName, res.FeedName, dbFeed.Url)

	return nil
}

func followingHandler(st *state, cmd command) error {
	err := loginHandler(st, command{
		cmd:  "login",
		args: []string{st.cfg.CurrentUserName},
	})

	// Check if user is logged in
	if st.userId == uuid.Nil {
		return fmt.Errorf("you must be logged in to see your following feeds")
	}

	results, err := st.DBQueries.GetFeedFollowsForUser(context.Background(), st.userId)
	if err != nil {
		return fmt.Errorf("there was an error fetching feed follows for user: %w", err)
	}

	if len(results) != 0 {
		fmt.Printf("%s is following:\n", results[0].UserName)
		for _, res := range results {
			fmt.Printf("%s (%s)\n", res.FeedName, res.FeedID)
		}
	}

	return nil
}

func unfollowHandler(st *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("feed name is required to unfollow a feed")
	}

	err := loginHandler(st, command{
		cmd:  "login",
		args: []string{st.cfg.CurrentUserName},
	})

	// Check if user is logged in
	if st.userId == uuid.Nil {
		return fmt.Errorf("you must be logged in to unfollow a feed")
	}

	feed, err := st.DBQueries.GetFeedByURL(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("there was an error fetching feed by url: %w", err)
	}

	err = st.DBQueries.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: st.userId,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("there was an error deleting feed follow: %w", err)
	}

	fmt.Printf("%s is no longer following %s (%s)\n", st.cfg.CurrentUserName, feed.Name, feed.Url)

	return nil
}

func browseHandler(st *state, cmd command) error {
	var limit int32 = 2

	if len(cmd.args) >= 1 {
		parsedLimit, err := fmt.Sscanf(cmd.args[0], "%d", &limit)
		if err != nil || parsedLimit <= 0 {
			return fmt.Errorf("invalid limit value: %s", cmd.args[0])
		}
	}
	err := loginHandler(st, command{
		cmd:  "login",
		args: []string{st.cfg.CurrentUserName},
	})

	if err != nil {
		return fmt.Errorf("you must be logged in to browse posts")
	}

	posts, err := st.DBQueries.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: st.userId,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("there was an error fetching posts for user: %w", err)
	}

	for _, post := range posts {
		fmt.Printf("%s\n%s\n\n", post.Title, post.Url)
	}

	return nil
}

func scrapeFeeds(st *state) {
	nextFeed, err := st.DBQueries.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Printf("Error fetching next feed to scrape: %v\n", err)
		return
	}

	err = st.DBQueries.MarkFeedFetched(context.Background(), database.MarkFeedFetchedParams{
		LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:            nextFeed.ID,
	})
	if err != nil {
		fmt.Printf("Error marking feed as fetched: %v\n", err)
		return
	}

	rssFeed, err := FetchFeed(context.Background(), nextFeed.Url)
	if err != nil {
		fmt.Printf("Error fetching feed: %v\n", err)
		return
	}

	for _, rssItem := range rssFeed.Channel.Item {
		// Parse the published date
		publishedAt, err := time.Parse(time.RFC1123Z, rssItem.PubDate)
		if err != nil {
			fmt.Printf("Error parsing publish date for post %q: %v\n", rssItem.Title, err)
			continue
		}

		// Create post entry
		_, err = st.DBQueries.CreatePost(context.Background(), database.CreatePostParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title:     rssItem.Title,
			Url:       rssItem.Link,
			Description: sql.NullString{
				String: rssItem.Descr,
				Valid:  rssItem.Descr != "",
			},
			PublishedAt: publishedAt,
			FeedID:      nextFeed.ID,
		})
		if err != nil {
			// Check if it's a duplicate URL constraint error
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
				// 23505 is the constraint violation code for UNIQUE constraints
				// Ignore duplicates - they'll happen a lot
				continue
			}
			// Log other errors
			fmt.Printf("Error creating post %q: %v\n", rssItem.Title, err)
		}
	}
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
	cmds.register("follow", followHandler)
	cmds.register("following", followingHandler)
	cmds.register("unfollow", unfollowHandler)
	cmds.register("browse", browseHandler)

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
