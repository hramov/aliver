package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"time"
)

const AppVersion = "0.0.1"

type App struct {
	Version string

	ClusterID  string        `yaml:"cluster_id"`
	InstanceID int           `yaml:"instance_id"`
	Ip         string        `yaml:"ip"`
	Mask       int           `yaml:"mask"`
	PortTCP    int           `yaml:"port_tcp"`
	PortUDP    int           `yaml:"port_udp"`
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

	cfg.App.Version = AppVersion

	return err
}
