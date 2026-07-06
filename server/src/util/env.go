package util

import (
	e "github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload" // autoload environment variables from .env
)

type Environment struct {
	HOST                    string  `env:"HOST"`
	PORT                    int     `env:"PORT"`
	DB_PATH                 string  `env:"DB_PATH"`
	MUSIC_PATH              string  `env:"MUSIC_PATH"`
	ICECAST_SERVER_HOST     string  `env:"ICECAST_SERVER_HOST"`
	ICECAST_SERVER_PORT     string  `env:"ICECAST_SERVER_PORT"`
	ICECAST_SERVER_PASSWORD string  `env:"ICECAST_SERVER_PASSWORD"`
	STREAM_BASE_URL         string  `env:"STREAM_BASE_URL"`
	PUBLIC_PASETO_KEY       string  `env:"PUBLIC_PASETO_KEY"`
	PRIVATE_PASETO_KEY      string  `env:"PRIVATE_PASETO_KEY"`
	MAX_SESSIONS            int     `env:"MAX_SESSIONS" envDefault:"50"`
	RATE_LIMIT_RPS          float64 `env:"RATE_LIMIT_RPS" envDefault:"1"`
	RATE_LIMIT_BURST        int     `env:"RATE_LIMIT_BURST" envDefault:"5"`
}

func LoadEnv() (Environment, error) {
	return e.ParseAs[Environment]()
}
