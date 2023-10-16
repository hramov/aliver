package config

import (
	"fmt"
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
	Interface  string `yaml:"interface"`
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

	for _, i := range ifaces {
		if i.Name == cfg.App.Interface {
			addrs, _ := i.Addrs()

			if len(addrs) == 0 {
				return fmt.Errorf("interface %s has no IP addresses", cfg.App.Interface)
			}

			if len(addrs) > 1 && cfg.App.Ip == nil {
				return fmt.Errorf("interface %s has multiple IP addresses and no IP address was specified", cfg.App.Interface)
			}

			if len(addrs) > 1 && cfg.App.Ip != nil {
				for _, addr := range addrs {
					v := addr.(*net.IPNet)
					if v.IP.String() == cfg.App.Ip.String() {
						ip = v.IP
						mask = v.Mask
					}
				}
			}

			if len(addrs) == 1 {
				v := addrs[0].(*net.IPNet)
				ip = v.IP
				mask = v.Mask
			}
		}
	}

	if ip == nil || mask == nil {
		return fmt.Errorf("ip address or mask is nil on interface %s\n", cfg.App.Interface)
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
