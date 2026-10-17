//go:build windows

package cmd

import "github.com/wiredoor/wiredoor-cli/utils"

func init() {
	serviceCmd.AddCommand(
		newServiceAction("install", "Install or update and start the service (administrator required)", utils.InstallService),
		newServiceAction("uninstall", "Stop and remove the service (administrator required)", func() error {
			return utils.DeleteService(utils.WiredoorServiceName)
		}),
		newServiceAction("restart", "Restart the installed service (administrator required)", utils.RestartService),
		newServiceAction("stop", "Stop the service (administrator required)", func() error {
			return utils.StopService(utils.WiredoorServiceName)
		}),
	)
}
