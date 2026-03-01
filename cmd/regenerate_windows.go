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

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
	"golang.org/x/sys/windows/svc"
)

var force = false

// regenerateCmd represents the regenerate command
var regenerateCmd = &cobra.Command{
	Use:   "regenerate",
	Short: "Regenerate this node's WireGuard keys and access token",
	Long: `Regenerate the WireGuard key pair and node access token.

This command securely generates a new WireGuard key pair and requests a new access token
from the Wiredoor server. It replaces the old credentials and updates the local config file.

Use this when:
  - You suspect your token or keys have been compromised
  - You need to rotate credentials for security compliance
  - You want to reset the node's identity with new keys

⚠️ Warning:
  Regenerating keys and token may cause a temporary downtime in all exposed services.
  The VPN tunnel will be restarted, and existing connections may be briefly interrupted.

Note:
  - This command requires a working connection and valid credentials
  - After regeneration, the old token and keys are invalidated

Examples:
  wiredoor regenerate
`,
	Example: `  # Regenerate keys and token for this node
  wiredoor regenerate`,
	Run: func(cmd *cobra.Command, args []string) {
		if !force {
			doContinue := false

			survey.AskOne(&survey.Confirm{
				Message: "This command may cause a temporary downtime in all exposed services. Continue?",
				Default: doContinue,
			}, &doContinue)

			if !doContinue {
				return
			}
		}
		utils.Terminal().StartProgress("Executing regenerate...")
		defer utils.Terminal().StopProgress()

		isWindowsService, err := svc.IsWindowsService()
		if err != nil {
			utils.Terminal().StopProgress()
			utils.Terminal().Errorf("to detect if running as service, %v\n", err)
			slog.Error(fmt.Sprintf("error detecting if I am a service, %v\n", err))
			os.Exit(1)
		}
		if isWindowsService {
			utils.Terminal().StopProgress()
			utils.Terminal().Errorf("regenerate command not usable as service")
			slog.Error("error, regenerate command not usable as service")
			os.Exit(1)
		}

		//2 send disconnect message

		//prepare data to send:
		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "regenerate"

		if resp, err := utils.ExecuteLocalSystemServiceTask(jsonToSend); err == nil {
			utils.Terminal().StopProgress()
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(resp, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					switch response {
					case "ok":
						wiredoor.Status()
						os.Exit(0)
					default:
						utils.Terminal().Errorf("Fail due to unhandled service reposnse: %v\n", response)
						slog.Error(fmt.Sprintf("unhandled service reposnse: %v", response))
						os.Exit(1)
					}
				} else {
					utils.Terminal().Errorf("Fail due to service reposnse format: %v\n", string(resp))
					slog.Error(fmt.Sprintf("response format error: %v", resp))
					os.Exit(1)
				}
			} else {
				utils.Terminal().Errorf("Fail due to service reposnse format: %v\n", string(resp))
				slog.Error(fmt.Sprintf("response format error: %v", resp))
				os.Exit(1)
			}
		} else {
			utils.Terminal().StopProgress()
			utils.Terminal().Errorf("Service communication error: %v\n", err)
			slog.Error(fmt.Sprintf("Service communication error: %v", err))
			os.Exit(1)
		}

		//TODO move to service ipc
		// wiredoor.Disconnect()

		// wiredoor.RegenerateKeys()
	},
}

func init() {
	rootCmd.AddCommand(regenerateCmd)
	regenerateCmd.Flags().BoolVarP(&force, "force", "f", false, "Force regenerate without confirmation")
}
