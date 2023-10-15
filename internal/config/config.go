package config

import (
	"gopkg.in/yaml.v3"
	"net"
	"os"
	"time"
)

const AppVersion = "0.0.1"

type App struct {
	Version string

	ClusterID  string `yaml:"cluster_id"`
	InstanceID int    `yaml:"instance_id"`
	Ip         net.IP
	Mask       net.IPMask
	Broadcast  net.IP
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

	ifaces, _ := net.Interfaces()
	var ip net.IP
	var mask net.IPMask

	// TODO set interface in config.yml
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				if v.IP.String() != "127.0.0.1" {
					ip = v.IP
					mask = v.Mask
				}
			}
		}
	}

	broadcast := net.ParseIP("0.0.0.0").To4()

	ip = ip.To4()

	for i := 0; i < len(ip); i++ {
		broadcast[i] = ip[i] | ^mask[i]
	}

	cfg.App.Ip = ip
	cfg.App.Mask = mask
	cfg.App.Broadcast = broadcast

	return err
}
