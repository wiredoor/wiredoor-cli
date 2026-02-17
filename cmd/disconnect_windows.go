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
	"golang.org/x/sys/windows/svc"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect this node from the Wiredoor server",
	Long: `Gracefully disconnect this node from the Wiredoor server and stop the VPN tunnel.

This command tears down the active WireGuard connection and disables access to all exposed services.
No configuration is deleted, so you can reconnect later using 'wiredoor connect'.

Typical use cases:
  - Temporarily stopping the connection
  - Restarting the tunnel
  - Preparing the node for maintenance

Note:
  This does NOT delete the node or token from the Wiredoor server. Use 'wiredoor disable' if you only want to stop a specific service.

Examples:
  wiredoor disconnect
  wiredoor disconnect && sleep 5 && wiredoor connect`,
	Run: func(cmd *cobra.Command, args []string) {

		isWindowsService, err := svc.IsWindowsService()
		if err != nil {
			log.Printf("error detecting if I am a service, %v\n", err)
			os.Exit(1)
		}
		if isWindowsService {
			log.Print("error, connect command not usable as service")
			os.Exit(1)
		}

		//2 send disconnect message

		//prepare data to send:
		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "disconnect"

		if resp, err := utils.ExecuteLocalSystemServiceTask(jsonToSend); err == nil {
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(resp, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					switch response {
					case "ok":
						fmt.Printf("Disconnected successfully.\n")
						os.Exit(0)
					default:
						fmt.Printf("Fail due to unhandled service response: %v\n", response)
						log.Printf("unhandled service response: %v", response)
						os.Exit(1)
					}
				} else {
					fmt.Printf("Fail due to service response format: %v\n", string(resp))
					log.Printf("response format error: %v", resp)
					os.Exit(1)
				}
			} else {
				fmt.Printf("Fail due to service response format: %v\n", string(resp))
				log.Printf("response format error: %v", resp)
				os.Exit(1)
			}
		} else {
			fmt.Printf("Service communication error: %v\n", err)
			log.Printf("Service communication error: %v", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}
