//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
	"golang.org/x/sys/windows/svc"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Establish a VPN connection to a Wiredoor server",
	Long: `Connect this node to a Wiredoor server and establish the VPN tunnel.

By default, this command reads the configuration from /etc/wiredoor/config.ini,
which includes the server URL and the node's authentication token.

This is the standard way to initiate a Wiredoor tunnel after the node has already been registered.

Optional flags:
  --url           Override the server URL defined in the config file
  --token         Override the node token defined in the config file
	--daemon        Enable Wiredoor daemon to keep the connection alive and allow remote control (default)
	--no-daemon     Disable automatic daemon startup after this command

Typical usage:
  - Run 'wiredoor connect' to connect using saved credentials
  - Use '--url' and '--token' to override settings for testing or manual connections`,
	Example: `  # Connect using saved configuration
  wiredoor connect

  # Override the server URL manually
  wiredoor connect --url https://wiredoor.example.com

  # Provide a custom token (e.g., for automation)
  wiredoor connect --token=ABCDEF123456`,

	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")
		token, _ := cmd.Flags().GetString("token")
		useDaemon, _ := cmd.Flags().GetBool("daemon")
		setDaemon := cmd.Flags().Changed("daemon")

		// _ = useDaemon
		_ = setDaemon

		isWindowsService, err := svc.IsWindowsService()
		if err != nil {

			fmt.Printf("error detecting if I am a service, %v\n", err)
			log.Printf("error detecting if I am a service, %v\n", err)
			os.Exit(1)
		}
		if isWindowsService {
			fmt.Print("error, connect command not usable as service")
			log.Print("error, connect command not usable as service")
			os.Exit(1)
		}

		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "connect"
		jsonToSend["url"] = url
		jsonToSend["token"] = token
		jsonToSend["daemon"] = useDaemon

		if resp, err := utils.ExecuteLocalSystemServiceTask(jsonToSend); err == nil {
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(resp, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					switch response {
					case "ok":
						wiredoor.Status()
						os.Exit(0)
					default:
						fmt.Printf("Fail due to unhandled service reposnse: %v", response)
						log.Printf("unhandled service reposnse: %v", response)
						os.Exit(1)
					}
				} else {
					fmt.Printf("Fail due to service reposnse format: %v", string(resp))
					log.Printf("response format error: %v", resp)
					os.Exit(1)
				}
			} else {
				fmt.Printf("Fail due to service reposnse format: %v", string(resp))
				log.Printf("response format error: %v", resp)
				os.Exit(1)
			}
		} else {
			log.Printf("Service comunication error: %v", err)
			os.Exit(1)
		}

		//check for admin

		// if !wiredoor.WireguardInterfaceExists() {
		// 	wiredoor.Connect(wiredoor.ConnectionConfig{URL: url, Token: token, UseDaemon: useDaemon, SetDaemon: setDaemon})
		// }
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().String("url", "", "Wiredoor server URL (optional, overrides config file)")
	connectCmd.Flags().String("token", "", "Node connection token (optional, overrides config file)")
	connectCmd.Flags().Bool("daemon", true, "Enable Wiredoor daemon mode (use --no-daemon to disable)")
}
