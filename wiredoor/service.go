package wiredoor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// StartService starts the wiredoor service based on init system
func StartService() error {
	init := getInitSystem()
	switch init {
	case "systemd":
		return run([]string{"systemctl", "start", "wiredoor.service"})
	case "openrc":
		return run([]string{"rc-service", "wiredoor", "start"})
	default:
		return fmt.Errorf("unsupported init system: %s", init)
	}
}

// StopService stops the wiredoor service
func StopService() error {
	init := getInitSystem()
	switch init {
	case "systemd":
		return run([]string{"systemctl", "stop", "wiredoor.service"})
	case "openrc":
		return run([]string{"rc-service", "wiredoor", "stop"})
	default:
		return fmt.Errorf("unsupported init system: %s", init)
	}
}

// RestartService restarts the wiredoor service
func RestartService() error {
	init := getInitSystem()
	switch init {
	case "systemd":
		return run([]string{"systemctl", "restart", "wiredoor.service"})
	case "openrc":
		return run([]string{"rc-service", "wiredoor", "restart"})
	default:
		return fmt.Errorf("unsupported init system: %s", init)
	}
}

// EnableService enables the wiredoor service to start on boot
func EnableService() error {
	init := getInitSystem()
	switch init {
	case "systemd":
		return run([]string{"systemctl", "enable", "wiredoor.service"})
	case "openrc":
		return run([]string{"rc-update", "add", "wiredoor", "default"})
	default:
		return fmt.Errorf("unsupported init system: %s", init)
	}
}

// DisableService disables the wiredoor service from starting on boot
func DisableService() error {
	init := getInitSystem()
	switch init {
	case "systemd":
		return run([]string{"systemctl", "disable", "wiredoor.service"})
	case "openrc":
		return run([]string{"rc-update", "del", "wiredoor"})
	default:
		return fmt.Errorf("unsupported init system: %s", init)
	}
}

// getDistroID detects the OS from /etc/os-release
func getDistroID() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.SplitN(line, "=", 2)[1], `"`)
		}
	}
	return ""
}

// getInitSystem returns "systemd" or "openrc"
func getInitSystem() string {
	switch getDistroID() {
	case "alpine":
		return "openrc"
	default:
		return "systemd"
	}
}

// run executes a command and prints output
func run(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
