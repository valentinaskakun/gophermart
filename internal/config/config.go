package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog"
)

type Config struct {
	Address       string `env:"ADDRESS"`
	StoreInterval string `env:"STORE_INTERVAL"`
	StoreFile     string `env:"STORE_FILE"`
	Key           string `env:"KEY"`
	Database      string `env:"DATABASE_DSN"`
	Restore       bool   `env:"RESTORE"`
}

func LoadConfigServer() (config Config, err error) {
	log := zerolog.New(os.Stdout)
	flag.StringVar(&config.Address, "a", ":8080", "")
	flag.StringVar(&config.Database, "d", "postgres://postgres:postgrespw@localhost:55000", "")
	flag.StringVar(&config.Key, "k", "", "")
	flag.Parse()
	err = env.Parse(&config)
	if err != nil {
		log.Warn().Msg(err.Error())
	}
	return
}

func Hash(msg string, key string) (hash string) {
	src := []byte(msg)
	h := hmac.New(sha256.New, []byte(key))
	h.Write(src)
	hash = hex.EncodeToString(h.Sum(nil))
	return
}
