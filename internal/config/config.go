package config

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

const Token = "token"

type ServerConfig struct {
	Address        string `env:"RUN_ADDRESS"`
	DSN            string `env:"DATABASE_URI"`
	AccrualAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	SecretKey      string
}

func LoadServerConfig() (conf ServerConfig, err error) {
	flag.StringVar(&conf.Address, "a", ":80", "")
	flag.StringVar(&conf.AccrualAddress, "r", ":8080", "")
	flag.StringVar(&conf.DSN, "d", "", "")
	flag.Parse()
	conf.SecretKey = Token
	err = env.Parse(&conf)
	if err != nil {
		return conf, err
	}
	return
}
