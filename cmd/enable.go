/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var enableTtl string

var enableCmd = &cobra.Command{
	Use:   "enable <type> <ID>",
	Short: "Re-enable a previously disabled Wiredoor service",
	Long: `Re-enable a previously disabled Wiredoor service.

This command restores public access to a service that was disabled using 'wiredoor disable'.
It does not require redefining the configuration — it simply reactivates the route on the Wiredoor gateway.

Arguments:
  <type>   The type of service to enable: "http" or "tcp"
  <ID>     The ID of the service to enable (check services ID with 'wiredoor status')

Optional flags:
	--ttl						 Time-to-live duration for the exposure (e.g., "30m", "1h", "2d").
                   Automatically disables the service after the specified duration.

Examples:
  wiredoor enable http 4
  wiredoor enable tcp 5

Note:
  The service must already exist and be currently disabled.
  If the service is already enabled, this command has no effect.`,
	Example: `  # Re-enable a public website
  wiredoor enable http 4

  # Re-enable a TCP service
  wiredoor enable tcp 5`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		serviceType := args[0]
		serviceId := args[1]

		if serviceType != "http" && serviceType != "tcp" {
			fmt.Println("❌ Invalid service type. Must be 'http' or 'tcp'.")
			return
		}
		fmt.Printf("Enabling %s service '%s'...\n", serviceType, serviceId)

		wiredoor.EnableServiceByType(wiredoor.EnableRequest{ ServiceType: serviceType, ID: serviceId, Ttl: enableTtl })
	},
}

func init() {
	rootCmd.AddCommand(enableCmd)
	enableCmd.Flags().StringVar(&enableTtl, "ttl", "", "Time-to-live duration for the exposure (e.g., 30m, 1h, 2d)")
}
