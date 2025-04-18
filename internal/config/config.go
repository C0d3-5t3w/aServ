package config

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type Config struct {
	AppName  string `yaml:"app_name"`
	Port     string `yaml:"port"`
	LogLevel string `yaml:"log_level"`
	Auth     struct {
		Secret    string `yaml:"secret"`
		ExpireHrs int    `yaml:"expire_hrs"`
	} `yaml:"auth"`
	Storage struct {
		Path string `yaml:"path"`
	} `yaml:"storage"`
}

func LoadConfig() *Config {
	cfg := &Config{}

	configFile, err := ioutil.ReadFile("./pkg/config/config.yaml")
	if err != nil {
		log.Println("Warning: Could not read config file, using defaults:", err)
		return getDefaultConfig()
	}

	err = yaml.Unmarshal(configFile, cfg)
	if err != nil {
		log.Println("Warning: Could not parse config file, using defaults:", err)
		return getDefaultConfig()
	}

	return cfg
}

func getDefaultConfig() *Config {
	return &Config{
		AppName:  "aServ",
		Port:     "8080",
		LogLevel: "info",
		Auth: struct {
			Secret    string `yaml:"secret"`
			ExpireHrs int    `yaml:"expire_hrs"`
		}{
			Secret:    "default-secret-change-me",
			ExpireHrs: 24,
		},
		Storage: struct {
			Path string `yaml:"path"`
		}{
			Path: "./pkg/storage/storage.json",
		},
	}
}
