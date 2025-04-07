/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var disableCmd = &cobra.Command{
	Use:   "disable <type> <ID>",
	Short: "Temporarily disable an exposed Wiredoor service",
	Long: `Temporarily disable a Wiredoor service without deleting its configuration.

This command disables a previously exposed service (either HTTP or TCP),
preventing external access until it's re-enabled with 'wiredoor enable'.

It does not remove the service or its configuration, so it can be restored at any time.

Arguments:
  <type>   The type of service to disable: "http" or "tcp"
  <ID>     The ID of the service to disable (check services ID with 'wiredoor status')

Examples:
  wiredoor disable http 4
  wiredoor disable tcp 5

Use 'wiredoor enable' to re-enable the service later.`,
	Example: `  # Disable a public website temporarily
  wiredoor disable http 4

  # Disable a TCP-exposed database service
  wiredoor disable tcp 5`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		serviceType := args[0]
		serviceId := args[1]

		if serviceType != "http" && serviceType != "tcp" {
			fmt.Println("❌ Invalid service type. Must be 'http' or 'tcp'.")
			return
		}

		fmt.Printf("Disabling %s service '%s'...\n", serviceType, serviceId)

		wiredoor.DisableServiceByType(serviceType, serviceId)
	},
}

func init() {
	rootCmd.AddCommand(disableCmd)
}
