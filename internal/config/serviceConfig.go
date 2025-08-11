package config

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
)

type GatewayConfig struct {
	Port int `toml:"port"`
}

func GetGatewayConfig() (*GatewayConfig, error) {
	path := fmt.Sprintf("%s%s", GetConfigPath(), SERVICE_CONFIGNAME)

	blob, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}
	var conf GatewayConfig
	_, err = toml.Decode(string(blob), &conf)

	return &conf, err

}
