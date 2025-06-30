package config

import (
	"os"

	"github.com/dotenv-org/godotenvvault"
)

// Config struct, holds all config settings
// should be in main.go like the others, but nope
// say it deserves its own file
type Config struct {
	DatabaseURL Database
	APIKeys     API
	BotTokens   Bot
	Server      Server
	Env         Environ
}

type Database struct {
	URL string
}

type API struct {
	TMDBKey string
}

type Bot struct {
	DiscordToken  string
	TelegramToken string
}

type Server struct {
	Port string
	Host string
}

type Environ struct {
	Development string
	Staging     string
	Production  string
}

// Load Configuration in main.go
func Load() (*Config, error) {
	godotenvvault.Load()

	return &Config{
		DatabaseURL: Database{
			URL: os.Getenv("DATABASE_URL"),
		},
		APIKeys: API{
			TMDBKey: os.Getenv("TMDB_API_KEY"),
		},
		BotTokens: Bot{
			DiscordToken:  os.Getenv("DISCORD_BOT_TOKEN"),
			TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		},
		Server: Server{
			Port: os.Getenv("PORT"),
			Host: os.Getenv("HOST"),
		},
		Env: Environ{
			Development: os.Getenv("DEV"),
			Staging:     os.Getenv("STAGE"),
			Production:  os.Getenv("PROD"),
		},
	}, nil
}
