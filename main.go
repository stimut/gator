package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stimut/gator/internal/database"
	"github.com/stimut/gator/internal/rss"

	"github.com/stimut/gator/internal/config"
)

type state struct {
	db  *database.Queries
	cfg *config.Config
}

type command struct {
	name string
	args []string
}

type commands struct {
	commandMap map[string]func(*state, command) error
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commandMap[name] = f
}

func (c *commands) run(s *state, cmd command) error {
	if s == nil {
		return fmt.Errorf("nil state")
	}
	if c.commandMap[cmd.name] == nil {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}

	return c.commandMap[cmd.name](s, cmd)
}

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return fmt.Errorf("failed to get current user: %w", err)
		}

		return handler(s, cmd, user)
	}
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected no arguments")
	}

	_, err := s.db.Reset(context.Background())
	return err
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("single username argument expected with login command")
	}

	usr, err := s.db.GetUserByName(context.Background(), cmd.args[0])
	if err != nil {
		return err
	}

	s.cfg.SetUser(usr.Name)
	fmt.Printf("Logged in as %s\n", cmd.args[0])

	return nil
}

func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("single username argument expected with login command")
	}

	usr, err := s.db.CreateUser(
		context.Background(),
		database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.args[0]})
	if err != nil {
		return err
	}

	s.cfg.SetUser(usr.Name)
	fmt.Printf("Created user: %s\n", usr.Name)
	fmt.Println(usr)

	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected no arguments")
	}

	users, err := s.db.GetAllUsers(context.Background())
	if err != nil {
		return err
	}

	for _, usr := range users {
		if usr.Name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", usr.Name)
		} else {
			fmt.Printf("* %s\n", usr.Name)
		}
	}

	return nil
}

func handleAddFeed(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("expected name and url arguments")
	}

	feed, err := s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:     uuid.New(),
			Name:   cmd.args[0],
			Url:    cmd.args[1],
			UserID: user.ID,
		})
	if err != nil {
		return err
	}

	_, err = s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:     uuid.New(),
			UserID: user.ID,
			FeedID: feed.ID,
		})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %w", err)
	}

	fmt.Printf("Added feed %s\n", cmd.args[0])

	return nil
}

func handleFeeds(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected no arguments")
	}

	feeds, err := s.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	for _, feed := range feeds {
		user, err := s.db.GetUserById(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("failed to get feed-owning user: %w", err)
		}

		fmt.Printf("* %s: %s (%s)\n", feed.Name, feed.Url, user.Name)
	}

	return nil
}

func handleFollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected url of feed to follow")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	follow, err := s.db.CreateFeedFollow(
		context.Background(),
		database.CreateFeedFollowParams{
			ID:     uuid.New(),
			UserID: user.ID,
			FeedID: feed.ID,
		})
	if err != nil {
		return fmt.Errorf("failed to follow feed: %w", err)
	}

	fmt.Printf("%s is now following %s\n", follow.UserName, follow.FeedName)

	return nil
}

func handleUnfollow(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected url of feed to unfollow")
	}

	feed, err := s.db.GetFeedByUrl(context.Background(), cmd.args[0])
	if err != nil {
		return fmt.Errorf("failed to get feed: %w", err)
	}

	err = s.db.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{UserID: user.ID, FeedID: feed.ID})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %w", err)
	}

	fmt.Printf("%s is no longer following %s\n", user.Name, feed.Name)

	return nil
}

func handleFollowing(s *state, cmd command, user database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected no arguments")
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get follows for user: %w", err)
	}

	fmt.Printf("%s is following:\n", user.Name)
	for _, follow := range follows {
		fmt.Printf("* %s\n", follow.FeedName)
	}

	return nil
}

func handleBrowse(s *state, cmd command, user database.User) error {
	if len(cmd.args) > 1 {
		return fmt.Errorf("expected only 1 optional argument")
	}

	limit := 2
	if len(cmd.args) == 1 {
		var err error
		limit, err = strconv.Atoi(cmd.args[0])
		if err != nil {
			return fmt.Errorf("invalid limit provided: %v", cmd.args[0])
		}
	}

	posts, err := s.db.GetPostsForUser(
		context.Background(),
		database.GetPostsForUserParams{UserID: user.ID, Limit: int32(limit)})
	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	for _, post := range posts {
		fmt.Println(post.Title.String)
		fmt.Println()
		fmt.Println(post.Description.String)
		fmt.Println()
		fmt.Println()
		fmt.Println()
	}

	return nil
}

func handleAgg(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("expected argument for time between requests")
	}

	waitTime, err := time.ParseDuration(cmd.args[0])
	if err != nil {
		return fmt.Errorf("invalid argument for time between requests: %w", err)
	}

	fmt.Printf("Collecting feeds every %v\n\n", waitTime)
	ticker := time.NewTicker(waitTime)
	for ; ; <-ticker.C {
		err := scrapeNextFeed(s)
		if err != nil {
			return fmt.Errorf("failed to scrape feeds: %w", err)
		}
	}
}

func scrapeNextFeed(s *state) error {
	feed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return fmt.Errorf("failed to fetch next feed: %w", err)
	}

	if s.db.MarkFeedFetched(
		context.Background(),
		database.MarkFeedFetchedParams{
			ID:            feed.ID,
			LastFetchedAt: sql.NullTime{Time: time.Now(), Valid: true}}) != nil {
		return fmt.Errorf("failed to mark feed as fetched: %w", err)
	}

	contents, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		return fmt.Errorf("failed to fetch feed: %w", err)
	}

	for _, it := range contents.Channel.Item {

		var pubDate time.Time
		if it.PubDate != "" {
			pubDate, err = time.Parse(time.RFC3339, it.PubDate)
			if err != nil {
				pubDate, err = time.Parse(time.RFC1123Z, it.PubDate)
				if err != nil {
					return fmt.Errorf("failed to parse pub date: %w", err)
				}
			}
		}

		_, err = s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			Title:       sql.NullString{String: it.Title, Valid: it.Title != ""},
			Url:         sql.NullString{String: it.Link, Valid: it.Link != ""},
			Description: sql.NullString{String: it.Description, Valid: it.Description != ""},
			PublishedAt: sql.NullTime{Time: pubDate, Valid: !pubDate.IsZero()},
			FeedID:      feed.ID,
		})
		if err != nil {
			var perr *pq.Error
			if errors.As(err, &perr) {
				if perr.Code == "23505" {
					// unique violation -- ignore as will be url unique constraint
					continue
				}
			}
			return fmt.Errorf("failed to create post: %w", err)
		}
	}

	return nil
}

func main() {
	cfg := config.Read()
	c := commands{make(map[string]func(*state, command) error)}

	c.register("reset", handlerReset)

	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("users", handlerUsers)

	c.register("addfeed", middlewareLoggedIn(handleAddFeed))
	c.register("feeds", handleFeeds)
	c.register("follow", middlewareLoggedIn(handleFollow))
	c.register("unfollow", middlewareLoggedIn(handleUnfollow))
	c.register("following", middlewareLoggedIn(handleFollowing))

	c.register("browse", middlewareLoggedIn(handleBrowse))

	c.register("agg", handleAgg)

	a := os.Args
	if len(a) < 2 {
		fmt.Println("Command expected")
		os.Exit(1)
	}

	s := state{}
	s.cfg = &cfg

	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		log.Fatal(err)
	}
	dbQueries := database.New(db)
	s.db = dbQueries

	err = c.run(&s, command{a[1], a[2:]})
	if err != nil {
		log.Fatal(err)
	}
}
