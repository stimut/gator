package main

import (
	"fmt"

	"github.com/stimut/gator/internal/config"
)

type state struct {
	Config *config.Config
}

func main() {
	c := config.Read()
	c.SetUser("tim")

	updatedConfig := config.Read()
	fmt.Printf("%+v\n", updatedConfig)
}
