package main

import (
	"flag"
	"github.com/gammazero/nexus/client"
	"github.com/gieseladev/wampus"
	"github.com/micro/go-micro/config"
	"github.com/micro/go-micro/config/source/env"
	"github.com/micro/go-micro/config/source/file"
	"log"
)

type Config struct {
	DiscordToken string
	RouterURL    string

	Realm string
}

func loadConfig(path string) (*Config, error) {
	c := config.NewConfig()
	err := c.Load(
		file.NewSource(file.WithPath(path)),
		env.NewSource(env.WithStrippedPrefix("WAMPUS")),
	)

	if err != nil {
		return nil, err
	}

	var conf Config
	if err := c.Scan(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

func createWAMPConfig(conf *Config) client.Config {
	return client.Config{
		Realm: conf.Realm,
	}
}

func run() error {
	var configPath string

	flag.StringVar(&configPath, "config", "config.toml", "Config file location")
	flag.Parse()

	conf, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	c, err := wampus.Connect(conf.DiscordToken, conf.RouterURL, createWAMPConfig(conf))
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	return c.Open()
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
