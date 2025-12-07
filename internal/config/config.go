package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the configuration settings for the application.
type Config struct {
	DbURL           string `json:"db_url"`
	CurrentUserName string `json:"current_user_name"`
}

const configFileName = ".gatorconfig.json"

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, configFileName), nil
}

func write(cfg Config) error {
	configPath, err := getConfigFilePath()
	if err != nil {
		return err
	}
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	if err := encoder.Encode(cfg); err != nil {
		return err
	}
	return nil
}

//Read,  function that reads the JSON file found at ~/.gatorconfig.json and returns a Config struct. It should read the file from the HOME directory, then decode the JSON string into a new Config struct. I used os.UserHomeDir to get the location of HOME.

func Read() (*Config, error) {

	configPath, errF := getConfigFilePath()

	if errF != nil {
		return nil, errF
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SetUser method on the Config struct that writes the config struct to the JSON file after setting the current_user_name field.
func (c *Config) SetUser(userName string) error {
	c.CurrentUserName = userName

	return write(*c)
}
