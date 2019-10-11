package main

import (
	"context"
	"flag"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gieseladev/wampus"
	"github.com/micro/go-micro/config"
	"github.com/micro/go-micro/config/source"
	"github.com/micro/go-micro/config/source/env"
	"github.com/micro/go-micro/config/source/file"
	"log"
)

type DiscordConfig struct {
	Token string `json:"token"`
}

type WAMPConfig struct {
	URL   string `json:"url"`
	Realm string `json:"realm"`
}

type Config struct {
	Discord DiscordConfig `json:"discord"`
	WAMP    WAMPConfig    `json:"wamp"`
}

func loadConfig(path string) (*Config, error) {
	c := config.NewConfig()

	var sources []source.Source

	if path != "" {
		sources = append(sources, file.NewSource(file.WithPath(path)))
	}

	sources = append(sources, env.NewSource())

	err := c.Load(sources...)

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
		Realm: conf.WAMP.Realm,
	}
}

func run() error {
	var configPath string

	flag.StringVar(&configPath, "config", "", "Config file location")
	flag.Parse()

	conf, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	c, err := wampus.Connect(context.Background(), conf.Discord.Token, conf.WAMP.URL, createWAMPConfig(conf))
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	if err := c.Open(); err != nil {
		return err
	}

	<-c.Done()

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}
