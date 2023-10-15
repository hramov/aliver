package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

type App struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`

	ClusterID  string        `yaml:"cluster_id"`
	InstanceID int           `yaml:"instance_id"`
	Ip         string        `yaml:"ip"`
	Port       int           `yaml:"port"`
	Mode       string        `yaml:"mode"`
	Weight     int           `yaml:"weight"`
	Timeout    time.Duration `yaml:"timeout"`

	CheckScript   string        `yaml:"check_script"`
	CheckInterval time.Duration `yaml:"check_interval"`
	CheckRetries  int           `yaml:"check_retries"`
	CheckTimeout  time.Duration `yaml:"check_timeout"`

	RunScript  string        `yaml:"run_script"`
	RunTimeout time.Duration `yaml:"run_timeout"`

	StopScript  string        `yaml:"stop_script"`
	StopTimeout time.Duration `yaml:"stop_timeout"`
}

type Config struct {
	App *App
}

func LoadConfig(configPath string, cfg *Config) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, cfg)
	return err
}
