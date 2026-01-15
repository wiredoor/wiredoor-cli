//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
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

	log.Printf("Starting service, execute args: %v\n", args)
	s <- svc.Status{State: svc.StartPending}
	//running
	//channel for sync coms
	routineComs := make(chan struct{})
	//wait group for sync
	var waitGroupMonitor sync.WaitGroup
	// create go routine for monitoring
	log.Println("Begin monitoring routine")
	waitGroupMonitor.Add(1)
	go func() {

		defer waitGroupMonitor.Done()

		sleepSeconds := serviceInterval
		if sleepSeconds <= 0 {
			sleepSeconds = 10
		}
		timer := time.NewTimer(time.Second * time.Duration(sleepSeconds))
		defer timer.Stop()

		for {
			select {
			//when channel is closed
			case <-routineComs:
				log.Printf("Stop monitoring\n")
				//call deferred timer.stop
				return
			case <-timer.C:
				wiredoor.WatchHealt()
				//compatibility and stability
				if !timer.Stop() {
					<-timer.C
				}
				//wait 10 seconds before new check
				timer.Reset(time.Second * time.Duration(sleepSeconds))
			}
		}
	}()

	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	log.Printf("Start service\n")
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending} //notify status
				log.Printf("Stop service\n")
				//alert runing goroutine
				close(routineComs)
				//wait for cleanup
				log.Printf("Wait for cleanup\n")
				waitGroupMonitor.Wait()
				log.Printf("The end\n")
				return false, 0
			default: // never
			}
		}
		//never
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

  --serviceInterval   Interval in seconds to use with service command(default: 10)

Examples:

  # Watch status continuously
  wiredoor service --serviceInterval 10`,

	Run: func(cmd *cobra.Command, args []string) {

		isService, err := svc.IsWindowsService()

		if err != nil {
			log.Print("Unable to determine if running as service")
			log.Fatal(err)
		}
		if isService {
			logFileName := os.Getenv("PROGRAMDATA") + "\\WiredoorLastServiceLog.txt"
			logFile, err := os.Create(logFileName)
			if err == nil {
				defer logFile.Close()
				log.SetOutput(logFile)
			} else {
				//never
				log.Println("Warinig:Fail to create log file")
			}
			err = svc.Run(wiredoor.WiredoorServiceName, &wiredoorWindowsService{})
			if err != nil {
				log.Print("Fail to start service mode\n")
				os.Exit(1)
			}
		} else {
			log.Print("Running as console app, made for run as service ...\n")
			os.Exit(1)
		}
		// }
	},
}

func init() {
	rootCmd.AddCommand(windowsServiceCmd)
	windowsServiceCmd.Flags().IntVar(&serviceInterval, "serviceInterval", 10, "Polling interval in seconds (used with service)")
}
