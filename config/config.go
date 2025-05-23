package config

import (
	"go-simpler.org/env"
	"log"
)

type ConfigSchema struct {
	DB struct {
		Host string `env:"HOST" default:"localhost"`
		Port int    `env:"PORT" default:"5432"`
		User string `env:"USER" default:"postgres"`
		Pass string `env:"PASS" default:"postgres"`
		Name string `env:"NAME" default:"postgres"`
	} `env:"DB_"`

	BasicAuthUsername string `env:"BASIC_AUTH_USERNAME" default:"admin"`
	BasicAuthPassword string `env:"BASIC_AUTH_PASSWORD" default:"admin"`

	ShortCodeLength int `env:"SHORT_CODE_LENGTH" default:"6"`
}

var Config = ConfigSchema{}

func init() {
	if err := env.Load(&Config, nil); err != nil {
		log.Fatal(err)
	}
}
