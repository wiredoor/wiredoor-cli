//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"golang.org/x/sys/windows/svc"
)

var (
	server string
	token  string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Write Wiredoor connection settings to the config file",
	Long: `Preconfigure this node with the Wiredoor server URL and node token.

This command writes the provided settings to /etc/wiredoor/config.ini
so the node can later connect using 'wiredoor connect'.

It is useful for:
  - Automating provisioning of nodes
  - Preparing a node before establishing a connection
  - Changing the server or token without reconnecting immediately

Note:
  This command does NOT connect to the server or establish the VPN tunnel.
  Use 'wiredoor connect' after configuring.

Examples:
  wiredoor config --url=https://wiredoor.example.com --token=ABCDEF123456

Afterwards, simply run:
  wiredoor connect`,
	Example: `  # Configure the Wiredoor server and token
  wiredoor config --url=https://wiredoor.example.com --token=ABCDEF123456

  # Then connect when ready
  wiredoor connect`,
	Run: func(cmd *cobra.Command, args []string) {
		isWindowsService, err := svc.IsWindowsService()
		if err != nil {
			utils.Terminal().Printf("Wiredoor service detection error: %v\n", err)
			slog.Error("Service detection error", "error", err)
			os.Exit(1)
		}
		if isWindowsService {
			slog.Error("error, config command not usable as service")
			os.Exit(1)
		}

		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "config"
		jsonToSend["url"] = server
		jsonToSend["token"] = token

		utils.Terminal().StartProgress("Saving config...")
		// IPC does not permit update progress in an easy way
		defer utils.Terminal().StopProgress()

		if resp, err := utils.ExecuteLocalSystemServiceTask(jsonToSend); err == nil {
			utils.Terminal().StopProgress()
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(resp, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					if strings.Contains(response, "Configuration saved to") {
						utils.Terminal().Println(response)
						os.Exit(0)
					} else {
						utils.Terminal().Printf("Error: %v", response)
						slog.Error(fmt.Sprintf("Error: %v", response))
						os.Exit(1)
					}
				} else {
					utils.Terminal().Printf("Service response format error or missing field: %v", string(resp))
					slog.Error(fmt.Sprintf("Service response format error or missing field: %v", resp))
					os.Exit(1)
				}
			} else {
				utils.Terminal().Printf("Fail due to service reposnse format: %v", string(resp))
				slog.Error(fmt.Sprintf("response format error: %v", resp))
				os.Exit(1)
			}
		} else {
			utils.Terminal().StopProgress()
			utils.Terminal().Printf("Service comunication error: %v\n", err)
			slog.Error(fmt.Sprintf("Service comunication error: %v", err))
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.Flags().StringVar(&server, "url", "", "Wiredoor server URL (required)")
	configCmd.Flags().StringVar(&token, "token", "", "Node authentication token (required)")

	_ = configCmd.MarkFlagRequired("url")
	_ = configCmd.MarkFlagRequired("token")

	rootCmd.Flags().SortFlags = false

	rootCmd.AddCommand(configCmd)
}
