/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
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
		fmt.Println("Saving Wiredoor config to", server)

		wiredoor.SaveServerConfig(server, token)

		fmt.Println("✅ Configuration saved to /etc/wiredoor/config.ini")
	},
}

func init() {
	configCmd.Flags().StringVar(&server, "url", "", "Wiredoor server URL (required)")
	configCmd.Flags().StringVar(&token, "token", "", "Node authentication token (required)")

	_ = configCmd.MarkFlagRequired("server")
	_ = configCmd.MarkFlagRequired("token")

	rootCmd.Flags().SortFlags = false

	rootCmd.AddCommand(configCmd)
}
