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
	utils.Terminal().Printf("[wiredoor] Installing Windows service...\n")

	running, err := utils.WiredoorServiceRunning()
	if err != nil {
		return fmt.Errorf("determine service status: %v", err)
	}
	if running {
		return fmt.Errorf("Wiredoor service is running; stop it first")
	}
	exists, err := utils.WiredoorServiceExists()
	if err != nil {
		return fmt.Errorf("error detecting existing service: %v", err)
	}
	if exists {
		err = utils.DeleteService(utils.WiredoorServiceName)
		if err != nil {
			return fmt.Errorf("error deleting existing Wiredoor service: %v", err)
		}
		time.Sleep(3 * time.Second)
	}

	utils.Terminal().Printf("[wiredoor] Starting service %s\n", utils.WiredoorServiceName)
	err = utils.CreateServiceFromThisExecutable(utils.WiredoorServiceName, "", "")
	if err != nil {
		return fmt.Errorf("error installing Wiredoor service: %v", err)
	}
	// }
	err = utils.StartService(utils.WiredoorServiceName)
	if err != nil {
		return fmt.Errorf("error starting wiredoor service: %v", err)
	}
	utils.Terminal().Printf("[wiredoor] %s installed and started successfully.\n", utils.WiredoorServiceName)
	return nil
}

var installCmd = &cobra.Command{
	Use:    "install",
	Hidden: true,
	Short:  "Install as a Windows service",
	Long:   `Internal use for the installer or installation repair.`,
	Example: `
  # Install this executable as a service for IPC
  wiredoor install`,
	Run: func(cmd *cobra.Command, args []string) {
		//generate a capable user for run service, remove

		// try install
		utils.Terminal().StartProgress(fmt.Sprintf("Installing as a service..."))
		defer utils.Terminal().StopProgress()
		if err := installService(); err != nil {
			utils.Terminal().StopProgress()
			utils.Terminal().Errorf("Installation error: %v\n", err)
		} else {
			utils.Terminal().StopProgress()
			utils.Terminal().Printf("Service installed...\n")
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
