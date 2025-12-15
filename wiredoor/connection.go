package wiredoor

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
	// "golang.org/x/sys/windows" //windows admin
)

// var interfaceName = "wg0"
var tunnelName = "wg0" //used to stop service
var configFilename = tunnelName + ".conf"

// system paths
var locationOfAPPDATA = os.Getenv("APPDATA")
var locationOfTEMP = os.Getenv("TEMP")

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

		// Using wireguard service
		manualWindowsConnect()
		fmt.Println("Waiting for connection starts (5 secs max)")

		//5 secs max
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if WireguardInterfaceExists() {
				break
			}
		}
		//last check
		Status()
	} else {
		fmt.Println("Error: Unable to connect we can't communicate with wiredoor server to get node configuration")
	}
}

func RestartTunnel() {
	manualWindowsRestart()
}

func Disconnect() {
	ensureRoot()
	manualWindowsDisconnect()
}

func ensureRoot() {

	adminCheck := exec.Command("net", "session")

	if err := adminCheck.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
		os.Exit(1)
	}
	// var token windows.Token
	// process := windows.CurrentProcess()
	// err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
	// defer token.Close()

	// var elevation windows.TokenElevation
	// var returnedLen uint32
	// err = windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(&elevation), uint32(unsafe.Sizeof(elevation)), &returnedLen)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
	// if elevation.TokenIsElevated == 0 {
	// 	fmt.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
}

func manualWindowsConnect() {
	config := GetNodeConfig()
	err := os.WriteFile(locationOfTEMP+"\\"+configFilename, []byte(config), 0600)
	if err != nil {
		log.Fatal(err)
	}
	//wireguard /installtunnelservice full_file_path
	up := exec.Command("wireguard", "/installtunnelservice", locationOfTEMP+"\\"+configFilename)

	fmt.Println("wireguard " + "/installtunnelservice " + locationOfTEMP + "\\" + configFilename)

	if IsDaemonEnabled() {
		fmt.Println("IGNORING DAEMON ...")
	}

	if err := up.Run(); err != nil {
		log.Fatal("Error: Unable to connect to tunnel")
	}

	//iniciar servicio
	//net start WireGuardTunnel$wg11
	start := exec.Command("net", "start", "WireGuardTunnel$"+tunnelName)
	if err := start.Run(); err != nil {
		log.Printf("Error: Unable to start tunnel service, %s\n", err.Error())
	}

	// if ExistWireguardConfigFile() {
	// 	_ = os.Remove(locationOfTEMP + "\\" + configFilename)
	// }
}

func manualWindowsRestart() {
	//net stop WireGuardTunnel$wg0
	stop := exec.Command("net", "stop", "WireGuardTunnel$"+tunnelName)
	if err := stop.Run(); err != nil {
		log.Fatal("Error: Unable to stop tunnel service")
	}
	//net start WireGuardTunnel$wg11
	start := exec.Command("net", "start", "WireGuardTunnel$"+tunnelName)
	if err := start.Run(); err != nil {
		log.Fatal("Error: Unable to start tunnel service")
	}
}

func manualWindowsDisconnect() {

	log.Println("Disconecting...")

	//net stop WireGuardTunnel$wg0
	stop := exec.Command("net", "stop", "WireGuardTunnel$"+tunnelName)
	if err := stop.Run(); err != nil {
		log.Fatal("Error: Unable to stop tunnel service")
	}

	//wireguard /uninstalltunnelservice wg0
	down := exec.Command("wireguard", "/uninstalltunnelservice", tunnelName)

	if IsDaemonEnabled() {
		StopService()
		DisableService()
	}

	if err := down.Run(); err != nil {
		log.Printf("Error: Unable to disconnect: %v", err)
	}

	if ExistWireguardConfigFile() {

		_ = os.Remove(locationOfTEMP + "\\" + configFilename)
	}
}

func ExistWireguardConfigFile() bool {
	_, err := os.Stat(locationOfTEMP + "\\" + configFilename)

	return err == nil
}
