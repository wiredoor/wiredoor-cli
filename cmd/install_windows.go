//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
)

func installService() error {
	fmt.Printf("[wiredoor] Installing windows service...\n")

	running, err := utils.WiredoorServiceRunning()
	if err != nil {
		return fmt.Errorf("determine service status: %v", err)
	}
	if running {
		return fmt.Errorf("wiredoor service is running, stop it first")
	}
	exists, err := utils.WiredoorServiceExists()
	if err != nil {
		return fmt.Errorf("error detecting old service : %v", err)
	}
	if exists {
		err = utils.DeleteService(utils.WiredoorServiceName)
		if err != nil {
			return fmt.Errorf("error when delete old wiredoor service: %v", err)
		}
		time.Sleep(3 * time.Second)
	}

	fmt.Printf("[wiredoor] Starting service %s\n", utils.WiredoorServiceName)
	err = utils.CreateServiceFromThisExecutable(utils.WiredoorServiceName, "", "")
	if err != nil {
		return fmt.Errorf("error instaling wiredoor service: %v", err)
	}
	// }
	err = utils.StartService(utils.WiredoorServiceName)
	if err != nil {
		return fmt.Errorf("error starting wiredoor service: %v", err)
	}
	fmt.Printf("[wiredoor] %s installed and started successfully.\n", utils.WiredoorServiceName)
	return nil
}

var installCmd = &cobra.Command{
	Use:    "install",
	Hidden: true,
	Short:  "Install as service on Windows",
	Long:   `Internal use, for installer or installation repair.`,
	Example: `
  # Install this executable as service for IPC
  wiredoor install`,
	Run: func(cmd *cobra.Command, args []string) {
		//generate a capable user for run service, remove

		// try install
		if err := installService(); err != nil {
			fmt.Printf("Installation error: %v\n", err)
		} else {
			fmt.Printf("Service installed...\n")
		}

	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
