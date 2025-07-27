package main

import (
	"os"
	"strconv"
)

type Config struct {
	Arrd        string
	CraftyAddr  string
	Port        int
	Key         string
	Timeout     int
}

var config *Config

func getConfig() Config {
	if config != nil {
		return *config
	}
	config = new(Config)
	config.Key = os.Getenv("CraftyKey")
	if config.Key == "" {
		panic("ProxyKey is not set, aborting...")
	}
	config.Addr = os.Getenv("ProxyAddr")
	config.CraftyAddr = os.Getenv("CraftyAddr")
	if config.Addr == "" {
		panic("ProxyAddr is not set, aborting...")
	}
	var err error
	config.Port, err = strconv.Atoi(os.Getenv("ProxyPort"))
	if err != nil {
		config.Port = 443
	}
	config.Timeout, err = strconv.Atoi(os.Getenv("ProxyTimeout"))
	if err != nil {
		config.Timeout = 5
	}
	return *config
}
