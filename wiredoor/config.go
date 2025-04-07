package wiredoor

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

var configFile = "/etc/wiredoor/config.ini"

var defaultConfig = map[string]map[string]string{
	"server": {
		"url":   "",
		"token": "",
		"path":  "",
	},
	"client": {
		"keepalive": "25",
	},
	"daemon": {
		"enabled": "false",
	},
}

type ServerConfig struct {
	Url   string
	Token string
	Path  string
}

type ClientConfig struct {
	KeepAlive string
}

type DaemonConfig struct {
	Enabled string
}

type Config struct {
	Server ServerConfig
	Client ClientConfig
	Daemon DaemonConfig
}

func SaveServerConfig(server string, token string) {
	cfg, err := getIniFile()

	if err != nil {
		log.Fatalf("Unable to get configuration file: %v", err)
	}

	cfg.Section("server").Key("url").SetValue(server)
	cfg.Section("server").Key("token").SetValue(token)

	cfg.SaveTo(configFile)
}

func SaveDaemonConfig(useDaemon bool) {
	cfg, err := getIniFile()

	if err != nil {
		log.Fatalf("Unable to get configuration file: %v", err)
	}
	cfg.Section("daemon").Key("enabled").SetValue(boolToString(useDaemon))

	cfg.SaveTo(configFile)
}

func IsDaemonEnabled() bool {
	config := getConfig()

	return parseBool(config.Daemon.Enabled)
}

func IsServerConfigSet() bool {
	config := getConfig()

	return config.Server.Url != "" && config.Server.Token != ""
}

func getConfig() Config {
	cfg, err := getIniFile()

	if err != nil {
		log.Fatalf("Unable to get configuration file: %v", err)
	}

	return Config{
		Server: ServerConfig{
			Url:   cfg.Section("server").Key("url").String(),
			Token: cfg.Section("server").Key("token").String(),
			Path:  cfg.Section("server").Key("path").String(),
		},
		Client: ClientConfig{
			KeepAlive: cfg.Section("client").Key("keepalive").String(),
		},
		Daemon: DaemonConfig{
			Enabled: cfg.Section("daemon").Key("enabled").String(),
		},
	}
}

func getIniFile() (*ini.File, error) {
	cfg, err := ini.Load(configFile)

	if err != nil {
		if os.IsNotExist(err) {
			return createDefaultConfigFile()
		}
	}

	return cfg, err
}

func createDefaultConfigFile() (*ini.File, error) {
	dir := filepath.Dir(configFile)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	cfg := ini.Empty()

	for section, keys := range defaultConfig {
		sec, _ := cfg.NewSection(section)
		for key, value := range keys {
			sec.NewKey(key, value)
		}
	}

	err := cfg.SaveTo(configFile)

	return cfg, err
}

func parseBool(val string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	return val == "1" || val == "true" || val == "yes" || val == "on"
}

func boolToString(val bool) string {
	if val {
		return "true"
	}
	return "false"
}
