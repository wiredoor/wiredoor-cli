/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
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

		if (!wiredoor.WireguardInterfaceExists()) {
			wiredoor.Connect(wiredoor.ConnectionConfig{URL: url, Token: token, UseDaemon: useDaemon, SetDaemon: setDaemon})
		}
	},
}

func init() {
	rootCmd.AddCommand(connectCmd)

	connectCmd.Flags().String("url", "", "Wiredoor server URL (optional, overrides config file)")
	connectCmd.Flags().String("token", "", "Node connection token (optional, overrides config file)")
	connectCmd.Flags().Bool("daemon", true, "Enable Wiredoor daemon mode (use --no-daemon to disable)")
}
