package wiredoor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

// var interfaceName = "wg0"
var configFilename = "wg0.conf"

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
	config := GetNodeConfig()
	err := os.WriteFile("/etc/wireguard/"+configFilename, []byte(config), 0600)
	if err != nil {
		log.Fatal(err)
	}
	up := exec.Command("bash", "-c", "wg-quick up wg0") // wg-quick down wg0 > /dev/null &2>1

	if IsDaemonEnabled() {
		RestartService()
		EnableService()
	}

	if err := up.Run(); err != nil {
		log.Fatal("Error: Unable to connect to tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
	}
}

func manualLinuxRestart() {
	restart := exec.Command("bash", "-c", "wg-quick down wg0 && wg-quick up wg0") // wg-quick down wg0 > /dev/null &2>1
	if err := restart.Run(); err != nil {
		log.Fatal("Error: Unable to restart the tunnel, please review your user permissions or if you are inside container ensure that you have added the capability NET_ADMIN")
	}
}

func manualLinuxDisconnect() {
	if ExistWireguardConfigFile() {
		log.Println("Disconecting...")
		down := exec.Command("bash", "-c", "wg-quick down wg0") // wg-quick down wg0 > /dev/null &2>1

		if IsDaemonEnabled() {
			StopService()
			DisableService()
		}

		if err := down.Run(); err != nil {
			log.Printf("Error: Unable to disconnect: %v", err)
		}
		_ = os.Remove("/etc/wireguard/" + configFilename)
	}
}

func ExistWireguardConfigFile() bool {
	_, err := os.Stat("/etc/wireguard/" + configFilename)

	return err == nil
}
