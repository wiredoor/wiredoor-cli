//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"golang.org/x/sys/windows/svc"
)

func installService() error {
	fmt.Printf("[wiredoor] Installing windows service...\n")
	// user, password, err := utils.CreateServiceAccount(utils.WIredoorServiceUserName)
	// if err != nil {
	// 	log.Printf("error creating dedicated user for service, using systemwide default user:\n%v\n", err)
	// }
	running, err := utils.WiredoorServiceRunning()
	if err != nil {
		return fmt.Errorf("determine service status: %v", err)
	}
	if running {
		return fmt.Errorf("Wiredoor service is running; stop it first")
	}
	exists, err := utils.WiredoorServiceExists()
	if err != nil {
		return fmt.Errorf("error detecting old service: %v", err)
	}
	if exists {
		err = utils.DeleteService(utils.WiredoorServiceName)
		if err != nil {
			return fmt.Errorf("error when delete old wiredoor service: %v", err)
		}
		time.Sleep(3 * time.Second)
	}
	// err = utils.CreateServiceFromThisExecutable(utils.WiredoorServiceName, user, password)
	// if err != nil {
	// 	//policy fail
	// 	log.Printf("Warninig, when install wiredoor service using alternate user, using default SYSTEM service user  : %v", err)
	// 	if err = utils.DeleteUser(user); err != nil {
	// 		log.Printf("Warninig, error on cleanup TMP user : %v", err)
	// 	}
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

type wiredoorInstallerService struct{}

func (wsvc *wiredoorInstallerService) Execute(args []string,
	r <-chan svc.ChangeRequest,
	s chan<- svc.Status) (bool, uint32) {
	log.Println("Installer service starting")
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running}
	err := installService()
	if err != nil {
		log.Printf("Install error: %v", err)
		return false, 1
	}
	log.Println("Installer finished successfully")
	s <- svc.Status{State: svc.StopPending}
	return false, 0
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
		isService, _ := svc.IsWindowsService()
		if isService {
			//run as service
			logFileName := os.Getenv("PROGRAMDATA") + "\\WiredoorInstallerLog.txt"
			logFile, err := os.Create(logFileName)
			if err == nil {
				defer logFile.Close()
				log.SetOutput(logFile)
			}
			err = svc.Run("InstallerSvc", &wiredoorInstallerService{})
			if err != nil {
				log.Print("Fail to start service mode\n")
				os.Exit(1)
			}
		} else {
			// try install
			if err := installService(); err != nil {
				log.Fatalf("Installation error: %v", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
