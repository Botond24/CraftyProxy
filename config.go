package main

import (
	"os"
	"strconv"
)

type Config struct {
	Addr       string
	CraftyAddr string
	Port       int
	Key        string
	Timeout    int
}

var config *Config

func getConfig() Config {
	if config != nil {
		return *config
	}
	config = new(Config)
	config.Key = os.Getenv("CraftyKey")
	if config.Key == "" {
		println("CraftyKey is not set, aborting...")
		os.Exit(1)
	}
	config.Addr = os.Getenv("ProxyAddr")
	config.CraftyAddr = os.Getenv("CraftyAddr")
	if config.CraftyAddr == "" {
		println("CraftyAddr is not set, aborting...")
		os.Exit(1)
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
