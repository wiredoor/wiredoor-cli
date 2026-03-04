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
			utils.Terminal().Printf("Wiredoor service detection error: %v\n", err)
			slog.Error("Service detection error", "error", err)
			os.Exit(1)
		}
		if isWindowsService {
			slog.Error("error, connect command not usable as service")
			os.Exit(1)
		}

		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "connect"
		jsonToSend["url"] = url
		jsonToSend["token"] = token
		jsonToSend["daemon"] = useDaemon

		utils.Terminal().StartProgress("Connecting...")
		// IPC does not permit update progress in a easy way
		defer utils.Terminal().StopProgress()

		if resp, err := utils.ExecuteLocalSystemServiceTask(jsonToSend); err == nil {
			utils.Terminal().StopProgress()
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(resp, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					switch response {
					case "ok":
						wiredoor.Status()
						os.Exit(0)
					case "Already Connected":
						wiredoor.Status()
						os.Exit(0)
					default:
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
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().String("url", "", "Wiredoor server URL (optional, overrides config file)")
	connectCmd.Flags().String("token", "", "Node connection token (optional, overrides config file)")
	connectCmd.Flags().Bool("daemon", true, "Enable Wiredoor daemon mode (use --no-daemon to disable)")
}
