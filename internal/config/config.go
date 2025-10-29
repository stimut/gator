package config

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DbUrl string `json:"db_url"`
	User  string `json:"current_user_name"`
}

func (c Config) SetUser(u string) {
	c.User = u
	err := write(c)
	if err != nil {
		log.Fatal(err)
	}
}

func Read() Config {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		log.Fatal(err)
	}

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		log.Fatal(err)
	}

	c := Config{}
	err = json.Unmarshal(data, &c)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

func write(c Config) error {
	configFilePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	err = os.WriteFile(configFilePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configFilePath := filepath.Join(homeDir, configFileName)

	return configFilePath, nil
}
