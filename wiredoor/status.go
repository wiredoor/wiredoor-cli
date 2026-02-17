package wiredoor

import (
	"fmt"
	"os"
	"time"

	"github.com/wiredoor/wiredoor-cli/utils"
)

func Status() {
	if !WireguardInterfaceExists() {
		utils.Terminal().Errorf("Wireguard interface %s is not active.", utils.TunnelName)
		utils.Terminal().Hint("Run 'wiredoor connect' to establish the tunnel.")
		return
	}

	if !CheckWiredoorServer(true) {
		utils.Terminal().Errorf("Tunnel seems active, but Wiredoor server unreachable.")
		utils.Terminal().Hint("Try running 'wiredoor connect' again or check server availability.")
		return
	}

	node := GetNode()
	// services := GetServices()

	printNodeInfoDetails(node)
}

func Health() {
	if !WireguardInterfaceExists() {
		utils.Terminal().Errorf("WireGuard interface " + utils.TunnelName + " is not active.")
		os.Exit(1)
		return
	}
	if !CheckWiredoorServer(true) {
		os.Exit(1)
		return
	}
	os.Exit(0)
}

func WatchHealt() {
	// log.Println("WatchHealt")
	if ExistWireguardConfigFile() {
		if !WireguardInterfaceExists() {
			node := GetNode()

			if node.Enabled {
				Connect(ConnectionConfig{})
			}
			return
		}

		if !CheckWiredoorServer(false) {
			RestartTunnel()
		}
	}
}

func WireguardInterfaceExists() bool {
	//OS implementation
	return interfaceExists()
}

func CheckWiredoorServer(debug bool) bool {
	ip := utils.LocalServerIP()

	if !utils.CheckPort(ip, 443) {
		return false
	} else {
		if debug {
			config := GetApiConfig()
			utils.Terminal().FinalizeProgress()
			utils.Terminal().Section("✔ Connection successful to: " + config.VPN_HOST)
		}
		return true
	}
}

func printNodeInfoDetails(node NodeInfo) {
	fmt.Println("")
	if node.IsGateway {
		if node.GatewayNetwork != "" && len(node.GatewayNetworks) == 0 {
			utils.Terminal().Printf("Using legacy gatewayNetwork field. Consider updating your Wiredoor Server.")

			utils.Terminal().KV("Gateway", fmt.Sprintf("%s (%s)", node.Name, node.Address))
			utils.Terminal().KV("Subnet", node.GatewayNetwork)
		}
		if len(node.GatewayNetworks) > 0 {
			var entries []string
			for _, net := range node.GatewayNetworks {
				if !utils.InterfaceExists(net.Interface) {
					utils.Terminal().Printf("⚠️ Interface \"%s\" does not exist on this system.\n", net.Interface)
				}

				entries = append(entries, fmt.Sprintf("%s: %s", net.Interface, net.Subnet))
			}

			utils.Terminal().KV("Gateway", fmt.Sprintf("%s (%s)", node.Name, node.Address))
			utils.Terminal().KV("Subnet", entries)
		}
	} else {
		utils.Terminal().KV("Node", fmt.Sprintf("%s (%s)", node.Name, node.Address))
	}
	fmt.Println("")
	utils.Terminal().KV("Handshake", formatRelativeTime(node.LatestHandshakeTimestamp))
	utils.Terminal().KV("TX", formatBytes(node.TransferTx))
	utils.Terminal().KV("RX", formatBytes(node.TransferRx))
	fmt.Println("")
	if len(node.HttpServices) > 0 || len(node.TcpServices) > 0 {
		utils.Terminal().Section("Services:")
		PrintHttpServices(node.HttpServices, node.IsGateway)
		PrintTcpServices(node.TcpServices, node.IsGateway)
	} else {
		utils.Terminal().Section("No services exposed yet.")
		utils.Terminal().Hint("Use 'wiredoor http' or 'wiredoor tcp' to expose a service.")
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatRelativeTime(ts int64) string {
	delta := time.Since(time.Unix(ts/1000, 0))
	switch {
	case delta < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(delta.Seconds()))
	case delta < time.Hour:
		if int(delta.Minutes()) > 1 {
			return fmt.Sprintf("%d minutes ago", int(delta.Minutes()))
		} else {
			return fmt.Sprintf("%d minute ago", int(delta.Minutes()))
		}
	case delta < 24*time.Hour:
		return fmt.Sprintf("%.1f hours ago", delta.Hours())
	default:
		return time.Unix(ts, 0).Format("2006-01-02 15:04")
	}
}
