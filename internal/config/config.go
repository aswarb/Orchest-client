package config

import (
	"fmt"
	"runtime"
)

const (
	APPNAME = "orchest"
	SERVICE_CONFIGNAME="services.toml"
)

func GetConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%APPDATA%/%s/",APPNAME)
	case "darwin":
		return fmt.Sprintf("~/Library/Application Support//%s/",APPNAME)
	case "linux":
		return fmt.Sprintf("~/.config/%s/",APPNAME)
	default:
		return fmt.Sprintf("~/%s/",APPNAME)
	}
}


