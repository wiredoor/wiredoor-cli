//go:build !windows
// +build !windows

package wiredoor

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"

	"github.com/wiredoor/wiredoor-cli/utils"
)

var configFilename = utils.TunnelName + ".conf"
var wireguardPath = "/etc/wireguard/"

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

	node := GetNode()

	if node.ID > 0 {
		nodeType := "node"

		if node.IsGateway {
			nodeType = "gateway"
		}

		fmt.Printf("Connecting %s %s...\n", nodeType, node.Name)

		// Using wg-quick
		manualLinuxConnect()

		Status()
	} else {
		fmt.Println("Error: Unable to connect we can't communicate with wiredoor server to get node configuration")
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
		fmt.Fprintln(os.Stderr, "Permission denied: root privileges are required (try with sudo)")
		os.Exit(1)
	}
}

func manualLinuxConnect() {
	if err := os.MkdirAll(wireguardPath, 0o700); err != nil {
		log.Fatalf(err.Error())
	}

	config := GetNodeConfig()

	err := os.WriteFile(wireguardPath+configFilename, []byte(config), 0600)
	if err != nil {
		log.Fatal(err)
	}
	up := exec.Command("bash", "-c", "wg-quick up "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1

	if IsDaemonEnabled() {
		RestartService()
		EnableService()
	}

	if err := up.Run(); err != nil {
		log.Fatal("Error: Unable to connect to tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
	}
}

func manualLinuxRestart() {
	restart := exec.Command("bash", "-c", "wg-quick down "+utils.TunnelName+" && wg-quick up "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1
	if err := restart.Run(); err != nil {
		log.Fatal("Error: Unable to restart the tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
	}
}

func manualLinuxDisconnect() {
	if ExistWireguardConfigFile() {
		log.Println("Disconecting...")
		down := exec.Command("bash", "-c", "wg-quick down "+utils.TunnelName) // wg-quick down wg0 > /dev/null &2>1

		if IsDaemonEnabled() {
			StopService()
			DisableService()
		}

		if err := down.Run(); err != nil {
			log.Printf("Error: Unable to disconnect: %v", err)
		}
		_ = os.Remove(wireguardPath + configFilename)
	}
}

func ExistWireguardConfigFile() bool {
	_, err := os.Stat(wireguardPath + configFilename)

	return err == nil
}

func getInterfaceName() (string, error) {
	out, err := exec.Command("wg", "show", "all", "dump").Output()
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
	iface, err := getInterfaceName()

	if err != nil || iface == "" {
		return false
	}

	_, netErr := net.InterfaceByName(iface)
	return netErr == nil
}
