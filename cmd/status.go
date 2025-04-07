/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"time"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var (
	checkHealth bool
	watch       bool
	interval    int
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check the current status of the Wiredoor node and its services",
	Long: `Displays the current connection status, including VPN status and exposed services.

By default, this command will print a summary of the current node configuration,
VPN status, and a list of exposed HTTP/TCP services.

Optional flags allow you to:
  --health     Run a simple health check (for CI or monitoring)
  --watch      Continuously monitor connection and service status
  --interval   Interval in seconds to use with --watch (default: 5)

Examples:
  # Check status once
  wiredoor status

  # Health check (for monitoring scripts or CI)
  wiredoor status --health

  # Watch status continuously
  wiredoor status --watch --interval 10`,
	Run: func(cmd *cobra.Command, args []string) {
		if checkHealth {
			wiredoor.Health()
			return
		}
		if watch {
			for {
				wiredoor.WatchHealt()
				sleepSeconds := interval
				if sleepSeconds <= 0 {
					sleepSeconds = 15
				}
				time.Sleep(time.Duration(sleepSeconds) * time.Second)
			}
		}
		wiredoor.Status()
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().BoolVar(&checkHealth, "health", false, "Perform a quick health check (useful for CI or monitoring)")
	statusCmd.Flags().BoolVar(&watch, "watch", false, "Continuously monitor connection status")
	statusCmd.Flags().IntVar(&interval, "interval", 10, "Polling interval in seconds (used with --watch)")
}
