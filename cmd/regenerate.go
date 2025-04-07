/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
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

		wiredoor.Disconnect()

		wiredoor.RegenerateKeys()
	},
}

func init() {
	rootCmd.AddCommand(regenerateCmd)
	regenerateCmd.Flags().BoolVarP(&force, "force", "f", false, "Force regenerate without confirmation")
}
