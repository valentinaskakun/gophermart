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

const TokenSecret = "secret256"

type Config struct {
	Address        string `env:"RUN_ADDRESS"`
	Database       string `env:"DATABASE_URI"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	KeyToken       string
}

func LoadConfigServer() (config Config, err error) {
	log := zerolog.New(os.Stdout)
	config.KeyToken = TokenSecret
	flag.StringVar(&config.Address, "a", "localhost:8080", "")
	flag.StringVar(&config.Database, "d", "postgres://postgres:postgrespw@localhost:55003", "")
	flag.StringVar(&config.AccrualAddress, "r", "", "")
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
