package config

import (
	"flag"
	"os"

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

func InitLog() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
}

func LoadConfigServer() (config Config, err error) {
	config.KeyToken = TokenSecret
	flag.StringVar(&config.Address, "a", "localhost:8080", "")
	flag.StringVar(&config.Database, "d", "postgres://postgres:postgrespw@localhost:55000", "")
	flag.StringVar(&config.AccrualAddress, "r", "localhost:8090", "")
	flag.Parse()
	err = env.Parse(&config)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "env.Parse(&config)",
		}).Error(err)
	}
	return
}
