package util

import (
	e "github.com/caarlos0/env/v11"
	_ "github.com/joho/godotenv/autoload" // autoload environment variables from .env
)

type Environment struct {
	HOST               string `env:"HOST"`
	PORT               int    `env:"PORT"`
	DB_PATH            string `env:"DB_PATH"`
	MUSIC_PATH         string `env:"MUSIC_PATH"`
	SPOTIFY_CLIENT_ID  string `env:"SPOTIFY_CLIENT_ID"`
	SPOTIFY_CLIENT_SECRET string `env:"SPOTIFY_CLIENT_SECRET"`
}

func Load() (Environment, error) {
	return e.ParseAs[Environment]()
}
