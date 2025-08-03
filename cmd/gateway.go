/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var gatewaySubnet, gatewayInterface string

var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Manage or configure the local Wiredoor gateway",
	Long: `Manage or configure the local Wiredoor gateway.

This command is used to update the subnet that the Wiredoor gateway uses to determine
internal network routing. The subnet must be provided in CIDR format and should match
your internal network or Kubernetes service subnet.

If no flags are provided, this command will show the help output.

Optional flags:
  --interface           Output interface for the gateway (default is "eth0" or default system interface)
  --subnet              Subnet in CIDR format (e.g., "10.42.0.0/16")

Examples:
  wiredoor gateway --subnet=10.42.0.0/16
  wiredoor gateway --subnet=172.30.0.0/24 --interface=eth0

Note:
  This command must be run from a registered gateway node with an active connection.`,
	Example: `  # Update the gateway subnet to match a Kubernetes service network
  wiredoor gateway --subnet=10.42.0.0/16`,
	Run: func(cmd *cobra.Command, args []string) {
		if gatewaySubnet == "" {
			cmd.Help()
			return
		}

		if gatewayInterface	== "" {
			gatewayInterface = utils.GetDefaultInterfaceName()
		}

		if _, _, err := net.ParseCIDR(gatewaySubnet); err != nil {
			fmt.Printf("❌ Invalid subnet format: %s\n", gatewaySubnet)
			return
		}

		fmt.Printf("Updating gateway subnet to '%s' using interface '%s'...\n", gatewaySubnet, gatewayInterface)

		wiredoor.UpdateGatewaySubnet(wiredoor.GatewayNetwork{ Interface: gatewayInterface, Subnet: gatewaySubnet })
	},
}

func init() {
	rootCmd.AddCommand(gatewayCmd)
	gatewayCmd.Flags().StringVar(&gatewayInterface, "interface", "", "Output interface for the gateway (default is \"eth0\")")
	gatewayCmd.Flags().StringVar(&gatewaySubnet, "subnet", "", "Subnet in CIDR format (e.g., 10.42.0.0/16)")
}
