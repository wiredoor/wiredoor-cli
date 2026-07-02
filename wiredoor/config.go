package wiredoor

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/wiredoor/wiredoor-cli/utils"
	"gopkg.in/ini.v1"
)

// !TODO search for dependencies on var configFile
var configFile = GetConfigLocation()

var defaultConfig = map[string]map[string]string{
	"profile": {
		"active": "default",
	},
	"profiles.default": {
		"url":   "",
		"token": "",
		"path":  "/",
	},
	"client": {
		"keepalive": "25",
	},
	"daemon": {
		"enabled": "false",
	},
}

type ServerConfig struct {
	Name  string
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
	ActiveProfile string
	Profiles      map[string]ServerConfig

	// legacy server compatibility
	Server ServerConfig

	Client ClientConfig
	Daemon DaemonConfig
}

func GetConfigLocation() string {
	currentOS := runtime.GOOS
	switch currentOS {
	case "windows":
		return os.Getenv("PROGRAMDATA") + "\\wiredoor\\config.ini"
	case "linux":
		return "/etc/wiredoor/config.ini"
	default:
		return "/etc/wiredoor/config.ini"
	}
}

func SaveServerConfig(server string, token string) error {
	cfg, err := getIniFile()

	if err != nil {
		utils.Terminal().Errorf("Unable to get configuration file: %v", err)
		return err
	}

	activeProfile := cfg.Section("profile").Key("active").String()
	if activeProfile == "" {
		activeProfile = "default"
		cfg.Section("profile").Key("active").SetValue(activeProfile)
	}

	sectionName := "profiles." + activeProfile
	sec := cfg.Section(sectionName)

	sec.Key("url").SetValue(server)
	sec.Key("token").SetValue(token)

	if strings.TrimSpace(sec.Key("path").String()) == "" {
		sec.Key("path").SetValue("/")
	}

	return cfg.SaveTo(configFile)
}

func SaveDaemonConfig(useDaemon bool) {
	cfg, err := getIniFile()

	if err != nil {
		utils.Terminal().Errorf("Unable to get configuration file: %v", err)
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
		utils.Terminal().Errorf("Unable to get configuration file: %v", err)
	}

	client := ClientConfig{
		KeepAlive: cfg.Section("client").Key("keepalive").String(),
	}

	daemon := DaemonConfig{
		Enabled: cfg.Section("daemon").Key("enabled").String(),
	}

	profiles := map[string]ServerConfig{}

	for _, section := range cfg.Sections() {
		name := section.Name()

		if strings.HasPrefix(name, "profiles.") {
			profileName := strings.TrimPrefix(name, "profiles.")

			profiles[profileName] = ServerConfig{
				Name:  profileName,
				Url:   section.Key("url").String(),
				Token: section.Key("token").String(),
				Path:  defaultPath(section.Key("path").String()),
			}
		}
	}

	activeProfile := cfg.Section("profile").Key("active").String()

	// Legacy format fallback: [server]
	legacyServer := ServerConfig{
		Name:  "default",
		Url:   cfg.Section("server").Key("url").String(),
		Token: cfg.Section("server").Key("token").String(),
		Path:  defaultPath(cfg.Section("server").Key("path").String()),
	}

	// If no profiles exist but legacy server exists, expose it as default profile.
	if len(profiles) == 0 && legacyServer.Url != "" && legacyServer.Token != "" {
		profiles["default"] = legacyServer
		activeProfile = "default"
	}

	if activeProfile == "" {
		activeProfile = "default"
	}

	activeServer := profiles[activeProfile]

	return Config{
		ActiveProfile: activeProfile,
		Profiles:      profiles,

		// Backward-compatible field.
		Server: activeServer,

		Client: client,
		Daemon: daemon,
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

func defaultPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	return path
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
