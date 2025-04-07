package wiredoor

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/wiredoor/wiredoor-cli/utils"
)

func Status() {
	if !WireguardInterfaceExists() {
		fmt.Println("❌ WireGuard interface 'wg0' is not active.")
		fmt.Println("Run 'wiredoor connect' to establish the tunnel.")
		return
	}

	if !CheckWiredoorServer(true) {
		fmt.Println("❌ Tunnel seems active, but Wiredoor server unreachable.")
		fmt.Println("Try running 'wiredoor connect' again or check server availability.")
		return
	}

	node := GetNode()
	// services := GetServices()

	printNodeInfoDetails(node)
}

func Health() {
	if !WireguardInterfaceExists() {
		fmt.Println("❌ WireGuard interface 'wg0' is not active.")
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
	cmd := exec.Command("ip", "link", "show", "wg0")
	return cmd.Run() == nil
}

func CheckWiredoorServer(debug bool) bool {
	ip := utils.LocalServerIP()

	config := GetApiConfig("https://" + ip)

	if config.VPN_HOST == "" {
		return false
	} else {
		if debug {
			fmt.Println(" ✔ Connection successful to:", config.VPN_HOST)
		}
		return true
	}
}

func printNodeInfoDetails(node NodeInfo) {
	fmt.Println("")
	if node.IsGateway {
		fmt.Printf("🛡️  Gateway: %s (%s) → 🌐 Subnet: %s\n", node.Name, node.Address, node.GatewayNetwork)
	} else {
		fmt.Printf("🖥️  Node: %s (%s)\n", node.Name, node.Address)
	}
	fmt.Println("")
	fmt.Printf("🔐 Handshake: %s | TX: %s | RX: %s\n",
		formatRelativeTime(node.LatestHandshakeTimestamp),
		formatBytes(node.TransferTx),
		formatBytes(node.TransferRx),
	)
	fmt.Println("")
	if len(node.HttpServices) > 0 || len(node.TcpServices) > 0 {
		fmt.Println("🌐 Services:")
		PrintHttpServices(node.HttpServices, node.IsGateway)
		PrintTcpServices(node.TcpServices, node.IsGateway)
	} else {
		fmt.Println("🌐 No services exposed yet.")
		fmt.Println("👉 Use 'wiredoor http' or 'wiredoor tcp' to expose a service.")
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
