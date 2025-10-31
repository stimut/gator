package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stimut/gator/internal/database"
	"github.com/stimut/gator/internal/rss"

	"github.com/stimut/gator/internal/config"
)

type state struct {
	db     *database.Queries
	config *config.Config
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

	s.config.SetUser(usr.Name)
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

	s.config.SetUser(usr.Name)
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
		if usr.Name == s.config.User {
			fmt.Printf("* %s (current)\n", usr.Name)
		} else {
			fmt.Printf("* %s\n", usr.Name)
		}
	}

	return nil
}

func handleAddFeed(s *state, cmd command) error {
	if len(cmd.args) != 2 {
		return fmt.Errorf("expected name and url arguments")
	}

	user, err := s.db.GetUserByName(context.Background(), s.config.User)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	_, err = s.db.CreateFeed(
		context.Background(),
		database.CreateFeedParams{
			ID:        uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Name:      cmd.args[0],
			Url:       cmd.args[1],
			UserID:    user.ID,
		})
	if err != nil {
		return err
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

func handleAgg(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("expected no arguments")
	}

	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Println(feed)

	return nil
}

func main() {
	cfg := config.Read()
	c := commands{make(map[string]func(*state, command) error)}

	c.register("reset", handlerReset)

	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("users", handlerUsers)

	c.register("addfeed", handleAddFeed)
	c.register("feeds", handleFeeds)

	c.register("agg", handleAgg)

	a := os.Args
	if len(a) < 2 {
		fmt.Println("Command expected")
		os.Exit(1)
	}

	s := state{}
	s.config = &cfg

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
