//go:build !windows
// +build !windows

package wiredoor

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/wiredoor/wiredoor-cli/utils"
)

var configFilename = utils.TunnelName + ".conf"
var wireguardPath = "/etc/wireguard/"
var interfaceNameFile = "/var/run/wiredoor/" + utils.TunnelName + "-interface"

type ConnectionConfig struct {
	URL       string
	Token     string
	UseDaemon bool
	SetDaemon bool
}

// type WireGuardConfig struct {
// 	PrivateKey wgtypes.Key
// 	Peers      []wgtypes.PeerConfig
// }

func Connect(connection ConnectionConfig) {
	ensureRoot()

	if connection.URL != "" && connection.Token != "" {
		SaveServerConfig(connection.URL, connection.Token)
	}

	if connection.SetDaemon {
		SaveDaemonConfig(connection.UseDaemon)
	}

	utils.Terminal().StartProgress("Connecting...")
	defer utils.Terminal().StopProgress()

	node := GetNode()

	if node.ID > 0 {
		nodeType := "node"

		if node.IsGateway {
			nodeType = "gateway"
		}

		utils.Terminal().UpdateProgress("Connecting " + nodeType + " " + node.Name)

		// Using wg-quick
		manualLinuxConnect()

		Status()
	}
}

func RestartTunnel() {
	manualLinuxRestart()
}

func Disconnect() {
	ensureRoot()
	manualLinuxDisconnect()
}

func ensureRoot() {
	if os.Geteuid() != 0 {
		utils.Terminal().Errorf("Permission denied: root privileges are required")
		utils.Terminal().Hint("Try running with sudo or as root")
		os.Exit(1)
	}
}

func manualLinuxConnect() {
	if err := os.MkdirAll(wireguardPath, 0o700); err != nil {
		utils.Terminal().Errorf("Error creating WireGuard directory: %v", err)
		os.Exit(1)
	}

	config := GetNodeConfig()

	err := os.WriteFile(wireguardPath+configFilename, []byte(config), 0600)
	if err != nil {
		utils.Terminal().Errorf("Error writing WireGuard configuration file: %v", err)
		os.Exit(1)
	}
	//take care of spaces
	up := exec.Command("bash", "-c", "wg-quick up "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1

	if IsDaemonEnabled() {
		RestartService()
		EnableService()
	}

	if err := up.Run(); err != nil {
		utils.Terminal().Errorf("Unable to connect to tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
		os.Exit(1)
	}

	iface, err := getInterfaceName()
	if err != nil || iface == "" {
		utils.Terminal().Errorf("Unable to determine the interface name after connecting")
		os.Exit(1)
	}

	if err := os.MkdirAll("/var/run/wiredoor", 0o644); err != nil {
		utils.Terminal().Errorf("Error creating Wiredoor runtime directory: %v", err)
		os.Exit(1)
	}

	err = os.WriteFile(interfaceNameFile, []byte(iface), 0644)
	if err != nil {
		utils.Terminal().Errorf("Error writing Wiredoor interface file: %v", err)
		os.Exit(1)
	}
}

func manualLinuxRestart() {
	restart := exec.Command("bash", "-c", "wg-quick down "+utils.TunnelName+" && wg-quick up "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1
	if err := restart.Run(); err != nil {
		utils.Terminal().Errorf("Unable to restart the tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
		os.Exit(1)
	}
}

// !TODO Integrate MAC OS
func manualLinuxDisconnect() {
	if ExistWireguardConfigFile() {
		utils.Terminal().StartProgress("Disconnecting...")
		defer utils.Terminal().StopProgress()
		down := exec.Command("bash", "-c", "wg-quick down "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1

		if IsDaemonEnabled() {
			StopService()
			DisableService()
		}

		if err := down.Run(); err != nil {
			utils.Terminal().Errorf("Unable to disconnect: %v", err)
		}
		utils.Terminal().FinalizeProgress()
		utils.Terminal().Printf("Disconnected successfully.")

		_ = os.Remove(wireguardPath + configFilename)
		_ = os.Remove(interfaceNameFile)
	} else {
		utils.Terminal().Printf("No active WireGuard configuration found. Already disconnected.")
	}
}

func ExistWireguardConfigFile() bool {
	_, err := os.Stat(wireguardPath + configFilename)

	return err == nil
}

func getInterfaceName() (string, error) {
	if runtime.GOOS == "linux" {
		return utils.TunnelName, nil
	}
	out, err := exec.Command("sudo", "wg", "show", "all", "dump").Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("wg show all dump failed: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("wg show all dump exec: %w", err)
	}

	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) == 0 || strings.TrimSpace(fields[0]) == "" {
			return "", fmt.Errorf("wg dump: first line has no interface name")
		}
		return strings.TrimSpace(fields[0]), nil
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("scan wg dump: %w", err)
	}
	return "", fmt.Errorf("wg dump: no lines found")
}

func interfaceExists() bool {
	iface, err := os.ReadFile("/var/run/wiredoor/" + utils.TunnelName + "-interface")
	if err != nil || len(iface) == 0 {
		return false
	}

	ifaceName := strings.TrimSpace(string(iface))
	if ifaceName == "" {
		return false
	}

	_, netErr := net.InterfaceByName(ifaceName)
	return netErr == nil
}
