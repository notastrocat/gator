package config

import (
	"encoding/json"
	"io"
	"os"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DB_URL string `json:"db_url"`
	CurrentUserName string `json:"current_user_name,omitempty"`
}

func (cfg *Config) SetUser(name string) {
	cfg.CurrentUserName = name
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Failed to get user home directory: " + err.Error())
	}
	configPath := homeDir + "/" + configFileName

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		panic("Failed to marshal config data: " + err.Error())
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		panic("Failed to write config file: " + err.Error())
	}
}

// Reads the configuration from the config file and returns a deserialized Config struct. Panics if any error occurs during the process.
func Read() *Config {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic("Failed to get user home directory: " + err.Error())
	}
	configPath := homeDir + "/" + configFileName

	file, err := os.Open(configPath)
	if err != nil {
		panic("Failed to open config file: " + err.Error())
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		panic("Failed to read config file: " + err.Error())
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		panic("Failed to unmarshal config data: " + err.Error())
	}

	return &config
}
