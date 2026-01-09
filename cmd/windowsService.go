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
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
	"golang.org/x/sys/windows/svc"
)

type wiredoorWindowsService struct{}

func (wsvc *wiredoorWindowsService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {

	//log
	file, err := os.Create("Z:\\wiredoor\\wiredoorService.log")
	if err != nil {
		return true, 1
	}
	defer file.Close()
	// starting
	file.WriteString("Iniciando Servicio\n")
	s <- svc.Status{State: svc.StartPending}
	//running
	//channel for sync coms
	routineComs := make(chan struct{})
	//wait group for sync
	var waitGroupMonitor sync.WaitGroup
	// create go routine for monitoring
	//
	waitGroupMonitor.Add(1)
	go func() {

		defer waitGroupMonitor.Done()

		for {
			select {
			//when channel is closed
			case <-routineComs:
				file.WriteString("Rutina Cerrada...\n")
				return
			default:
				sleepSeconds := serviceInterval
				if sleepSeconds <= 0 {
					sleepSeconds = 15
				}
				wiredoor.WatchHealt()
				file.WriteString("Rutina ...\n")
				time.Sleep(time.Duration(sleepSeconds) * time.Second)
			}
		}
	}()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	file.WriteString("Iniciando Servicio\n")
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending} //notify status
				file.WriteString("Deteniendo servicio\n")
				//alert runing goroutine
				close(routineComs)
				//wait for cleanup
				file.WriteString("Esperando rutinas\n")
				waitGroupMonitor.Wait()
				file.WriteString("Terminado\n")
				return false, 0
			default: // nop
			}
		}
		//no
		time.Sleep(500 * time.Millisecond)
	}
}

var (
	// service         bool
	serviceInterval int
)

var windowsServiceCmd = &cobra.Command{
	Use:    "service",
	Hidden: true,
	Short:  "Check the current status on windows, as service",
	Long: `Check the current status (windows service only)
By default this command is for internal use, running wiredoor as windows service

Optional flags allowed:

  --interval   Interval in seconds to use with --watch (default: 5)

Examples:

  # Watch status continuously
  wiredoor service --serviceInterval 10`,

	Run: func(cmd *cobra.Command, args []string) {

		file, err := os.Create("Z:\\wiredoor\\wiredoorServiceApp.log")
		if err != nil {
			fmt.Print("-------\n")
			return
		}
		defer file.Close()

		file.WriteString("xxxxxxxxxxxxxxxxxxxxx\n")

		isService, err := svc.IsWindowsService()
		file.WriteString("Windows Service?\n")
		if err != nil {
			file.WriteString("Fail to determine if is a service\n")
			log.Fatal(err)
		}
		if isService {
			file.WriteString("Starting service\n")
			err = svc.Run(wiredoor.WiredoorServiceName, &wiredoorWindowsService{})
			if err != nil {
				file.WriteString("Fail to start service mode\n")
				os.Exit(1)
			}
		} else {
			file.WriteString("Running as common app, made for run as service ...\n")
			os.Exit(1)
		}
		// }
	},
}

func init() {
	rootCmd.AddCommand(windowsServiceCmd)

	// windowsServiceCmd.Flags().BoolVar(&service, "service", false, "Continuously monitor connection status as service")
	windowsServiceCmd.Flags().IntVar(&serviceInterval, "serviceInterval", 10, "Polling interval in seconds (used with --service)")
}
