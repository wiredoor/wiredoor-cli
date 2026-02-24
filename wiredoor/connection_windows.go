//go:build windows
// +build windows

package wiredoor

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
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
func ConnectApi(connection ConnectionConfig) error {
	// ensureRoot()

	if connection.URL != "" && connection.Token != "" {
		if err := SaveServerConfig(connection.URL, connection.Token); err != nil {
			return err
		}
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

		slog.Info(fmt.Sprintf("Connecting %s %s...", nodeType, node.Name))

		// Using wireguard service
		if err := manualWindowsConnect(); err != nil {
			return err
		}
		slog.Info("Waiting for connection (5 secs max)")

		// 5 secs max
		tunnelExists := false
		for i := 0; i < 10; i++ {
			time.Sleep(500 * time.Millisecond)
			if WireguardInterfaceExists() {
				tunnelExists = true
				break
			}
		}
		if !tunnelExists {
			return fmt.Errorf("wireguard interface not detected")
		}
		// if ExistWireguardConfigFile() {
		// 	_ = os.Remove(wireguardConfigFolder + configFilename)
		// }
		return nil
	} else {
		return fmt.Errorf("unable to connect we can't communicate with wiredoor server to get node configuration")
	}
}

func Connect(connection ConnectionConfig) {
	if err := ConnectApi(connection); err != nil {
		utils.Terminal().Printf("Connection error: %v", err)
		//!!!! DO NOT KILL SERVICE
		//===================================
		isSvc, err := svc.IsWindowsService()
		if err != nil {
			return
		}
		if isSvc {
			return
		}
		//====================================
		os.Exit(1)
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
		slog.Error("Permission denied: Admin privileges are required")
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

func manualWindowsConnect() error {
	config := GetNodeConfig()

	//cleanup
	exists, err := utils.ServiceExists("WireGuardTunnel$" + utils.TunnelName)
	if err != nil {
		slog.Warn("Unable to determine if tunnel service exists, assuming true", "error", err)
		exists = true
	}
	if exists {
		//sc stop WireGuardTunnel$wg0
		stop := exec.Command("sc", "stop", "WireGuardTunnel$"+utils.TunnelName)
		if err := stop.Run(); err != nil {
			slog.Error("Unable to stop tunnel service", "error", err)
		}

		//wireguard /uninstalltunnelservice wg0
		down := exec.Command("wireguard", "/uninstalltunnelservice", utils.TunnelName)
		if err := down.Run(); err != nil {
			slog.Error("Unable to disconnect wireguard tunnel", "error", err)
		}
	}

	err = os.WriteFile(wireguardConfigFolder+configFilename, []byte(config), 0600)
	if err != nil {
		return fmt.Errorf("error on write cfg,%v", err)
	}
	//wireguard /installtunnelservice full_file_path
	up := exec.Command("wireguard", "/installtunnelservice", wireguardConfigFolder+configFilename)

	if err := up.Run(); err != nil {
		return fmt.Errorf("unable to connect to tunnel")
	}
	/*
		time.Sleep(250 * time.Millisecond)
		//iniciar servicio
		running, err := utils.ServiceRunning("WireGuardTunnel$" + utils.TunnelName)
		if err != nil {
			running = false
		}
		if !running {
			//sc start WireGuardTunnel$wg0
			if err := utils.StartService("WireGuardTunnel$" + utils.TunnelName); err != nil {
				return fmt.Errorf("unable to start tunnel service sunner, %v", err)
			}
		}
	*/
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
	return nil
}

func manualWindowsRestart() {
	//sc stop WireGuardTunnel$wg0
	stop := exec.Command("sc", "stop", "WireGuardTunnel$"+utils.TunnelName)
	if err := stop.Run(); err != nil {
		slog.Warn("Unable to stop tunnel service", "error", err)
	}
	//sc start WireGuardTunnel$wg0
	start := exec.Command("sc", "start", "WireGuardTunnel$"+utils.TunnelName)
	if err := start.Run(); err != nil {
		slog.Warn("Unable to start tunnel service", "error", err)
	}
}

func manualWindowsDisconnect() {

	// log.Println("Disconecting...")

	exists, err := utils.ServiceExists("WireGuardTunnel$" + utils.TunnelName)
	if err != nil {
		slog.Warn("Unable to determine if tunnel service exists, assuming true", "error", err)
		exists = true
	}
	if exists {
		//sc stop WireGuardTunnel$wg0
		stop := exec.Command("sc", "stop", "WireGuardTunnel$"+utils.TunnelName)
		if err := stop.Run(); err != nil {
			slog.Warn("Unable to stop tunnel service", "error", err)
		}

		//wireguard /uninstalltunnelservice wg0
		down := exec.Command("wireguard", "/uninstalltunnelservice", utils.TunnelName)
		if err := down.Run(); err != nil {
			slog.Error("Unable to disconnect wireguard tunnel: ", "error", err)
		}
	}

	// amIservice, err := svc.IsWindowsService()
	// if err != nil {
	// 	amIservice = false
	// }
	// if IsDaemonEnabled() { //not used
	// 	if !amIservice {
	// 		utils.StopService(utils.WiredoorServiceName)
	// 		utils.DisableService(utils.WiredoorServiceName)
	// 	}
	// }

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
		slog.Error("error on list interface names", "error", err)
		return false
	}
}
