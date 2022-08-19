package config

import (
	"flag"

	"github.com/caarlos0/env/v6"
	log "github.com/sirupsen/logrus"
)

const TokenSecret = "secret256"

type Config struct {
	Address        string `env:"RUN_ADDRESS"`
	Database       string `env:"DATABASE_URI"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	KeyToken       string
}

func LoadConfigServer() (config Config, err error) {
	config.KeyToken = TokenSecret
	flag.StringVar(&config.Address, "a", "localhost:8090", "")
	flag.StringVar(&config.Database, "d", "postgres://postgres:postgrespw@localhost:55003", "")
	flag.StringVar(&config.AccrualAddress, "r", "localhost:8080", "")
	flag.Parse()
	err = env.Parse(&config)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "env.Parse(&config)",
		}).Error(err)
	}
	return
}
