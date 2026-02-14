//go:build windows
// +build windows

package wiredoor

import (
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wiredoor/wiredoor-cli/utils"
	"golang.org/x/sys/windows/svc"
	// "golang.org/x/sys/windows" //windows admin
)

// // var interfaceName = "wg0"
// var TunnelName = "Wiredoor_Tunnel" //used to stop service
var configFilename = utils.TunnelName + ".conf"

// system paths
var wireguardConfigFolder = os.Getenv("PROGRAMDATA") + "\\wiredoor\\"

type ConnectionConfig struct {
	URL       string
	Token     string
	UseDaemon bool
	SetDaemon bool
}

//	type WireGuardConfig struct {
//		PrivateKey wgtypes.Key
//		Peers      []wgtypes.PeerConfig
//	}

func Connect(connection ConnectionConfig) {
	// ensureRoot()

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

		log.Printf("Connecting %s %s...\n", nodeType, node.Name)

		// Using wireguard service
		manualWindowsConnect()
		// log.Println("Waiting for connection starts (5 secs max)")

		//5 secs max
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if WireguardInterfaceExists() {
				break
			}
		}

		// if ExistWireguardConfigFile() {
		// 	_ = os.Remove(wireguardConfigFolder + configFilename)
		// }
	} else {
		log.Println("Error: Unable to connect we can't communicate with wiredoor server to get node configuration")
	}
}

func RestartTunnel() {
	manualWindowsRestart()
}

func Disconnect() {
	// ensureRoot()
	manualWindowsDisconnect()
}

func ensureRoot() {

	adminCheck := exec.Command("net", "session")

	if err := adminCheck.Run(); err != nil {
		log.Println("Permission denied: Admin privileges are required")
		os.Exit(1)
	}
	// var token windows.Token
	// process := windows.CurrentProcess()
	// err := windows.OpenProcessToken(process, windows.TOKEN_QUERY, &token)
	// if err != nil {
	// 	log.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
	// defer token.Close()

	// var elevation windows.TokenElevation
	// var returnedLen uint32
	// err = windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(&elevation), uint32(unsafe.Sizeof(elevation)), &returnedLen)
	// if err != nil {
	// 	log.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
	// if elevation.TokenIsElevated == 0 {
	// 	fmt.Fprintln(os.Stderr, "Permission denied: Admin privileges are required")
	// 	os.Exit(1)
	// }
}

func manualWindowsConnect() {
	config := GetNodeConfig()
	err := os.WriteFile(wireguardConfigFolder+configFilename, []byte(config), 0600)
	if err != nil {
		log.Printf("error on write cfg,%v", err)
		return
	}
	//wireguard /installtunnelservice full_file_path
	up := exec.Command("wireguard", "/installtunnelservice", wireguardConfigFolder+configFilename)

	if err := up.Run(); err != nil {
		log.Fatal("Error: Unable to connect to tunnel")
	}

	//iniciar servicio
	//sc start WireGuardTunnel$wg0
	//!TODO Move to native service api
	start := exec.Command("sc", "start", "WireGuardTunnel$"+utils.TunnelName)
	if err := start.Run(); err != nil {
		errorStr := err.Error()
		if strings.Contains(errorStr, "1056") {
			log.Printf("Tunnel service is running\n")
		} else {
			log.Printf("WARNING: Unable to start tunnel service sunner, %s\n", errorStr)
		}
	}
	amIservice, err := svc.IsWindowsService()
	if err != nil {
		amIservice = false
	}
	//!TODO clean, not used on new service mode api
	if IsDaemonEnabled() {
		if !amIservice {
			utils.StartService(utils.WiredoorServiceName)
		}
	}
}

func manualWindowsRestart() {
	//sc stop WireGuardTunnel$wg0
	stop := exec.Command("sc", "stop", "WireGuardTunnel$"+utils.TunnelName)
	if err := stop.Run(); err != nil {
		log.Println("Warinig: Unable to stop tunnel service")
	}
	//sc start WireGuardTunnel$wg0
	start := exec.Command("sc", "start", "WireGuardTunnel$"+utils.TunnelName)
	if err := start.Run(); err != nil {
		log.Println("Warinig: Unable to start tunnel service")
	}
}

func manualWindowsDisconnect() {

	log.Println("Disconecting...")

	exists, err := utils.ServiceExists("WireGuardTunnel$" + utils.TunnelName)
	if err != nil {
		log.Printf("Warning, unable to determine if tunnel service exists, assuming true : %v", err)
		exists = true
	}
	if exists {
		//sc stop WireGuardTunnel$wg0
		stop := exec.Command("sc", "stop", "WireGuardTunnel$"+utils.TunnelName)
		if err := stop.Run(); err != nil {
			log.Printf("Warnig: Unable to stop tunnel service: %v \n", err)
		}

		//wireguard /uninstalltunnelservice wg0
		down := exec.Command("wireguard", "/uninstalltunnelservice", utils.TunnelName)
		if err := down.Run(); err != nil {
			log.Printf("Error: Unable to disconnect wireguard tunnel: %v", err)
		}
	}

	amIservice, err := svc.IsWindowsService()
	if err != nil {
		amIservice = false
	}
	if IsDaemonEnabled() { //not used
		if !amIservice {
			utils.StopService(utils.WiredoorServiceName)
			utils.DisableService(utils.WiredoorServiceName)
		}
	}

	if ExistWireguardConfigFile() {
		_ = os.Remove(wireguardConfigFolder + configFilename)
	}
}

func ExistWireguardConfigFile() bool {
	// log.Printf("Wireguard cfg: %s", wireguardConfigFolder+configFilename)
	_, err := os.Stat(wireguardConfigFolder + configFilename)
	return err == nil
}

func interfaceExists() bool {
	// netsh interface show interface wg11
	// cmd := exec.Command("netsh", "interface", "show", "interface", utils.TunnelName) //wg0
	// err := cmd.Run()
	// if err != nil {
	// 	// log.Printf("Wireguard interface does not exist, %v", err)
	// 	return false
	// }
	// return true

	//!! move to internal api, netsh needs Admin priv

	if interfaces, err := net.Interfaces(); err == nil {
		for i := 0; i < len(interfaces); i++ {
			if interfaces[i].Name == utils.TunnelName /*&& (interfaces[i].Flags&net.FlagUp != 0) */ {
				return true
			}
		}
		return false
	} else {
		log.Printf("error on list interface names: %v", err)
		return false
	}
}
