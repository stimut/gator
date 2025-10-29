package main

import (
	"fmt"
	"log"
	"os"

	"github.com/stimut/gator/internal/config"
)

type state struct {
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
	if c.commandMap[cmd.name] == nil {
		return fmt.Errorf("unknown command: %s", cmd.name)
	}

	return c.commandMap[cmd.name](s, cmd)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) != 1 {
		return fmt.Errorf("single username argument expected with login command")
	}
	if s == nil {
		return fmt.Errorf("nil state")
	}

	s.config.SetUser(cmd.args[0])
	fmt.Printf("Logged in as %s\n", cmd.args[0])

	return nil
}

func main() {
	cfg := config.Read()
	c := commands{make(map[string]func(*state, command) error)}

	c.register("login", handlerLogin)

	a := os.Args
	if len(a) < 2 {
		fmt.Println("Command expected")
		os.Exit(1)
	}

	s := state{&cfg}
	err := c.run(&s, command{a[1], a[2:]})
	if err != nil {
		log.Fatal(err)
	}
}
