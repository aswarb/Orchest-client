package config

import (
	"fmt"
	"os"
)

const (
	APPNAME            = "orchest"
	SERVICE_CONFIGNAME = "services.toml"
)

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()

	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", configDir, APPNAME), nil

}
