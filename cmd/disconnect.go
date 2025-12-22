/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
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
		//check for admin
		if !utils.IsRoot() {
			//run as admin on windows or print a message on linux
			if err := utils.RelaunchAsRoot(); err == nil {
				// wiredoor.Status()
				os.Exit(0)
			}
		}
		//! TODO Continue anyway ?
		wiredoor.Disconnect()
	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}
