package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type (
	Config struct {
		Gateway  Gateway
		Services map[string]Service
	}

	Gateway struct {
		Port int `toml:"port"`
	}

	Service struct {
		Port int `toml:"port"`
	}
)

func GetGatewayConfig() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("%s/%s", configPath, SERVICE_CONFIGNAME)

	blob, err := os.ReadFile(path)
	fmt.Println(err)
	if err != nil {
		return nil, err
	}
	var conf Config
	_, err = toml.Decode(string(blob), &conf)

	return &conf, err

}
