//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
package cmd

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

/*
command format:
`

	{
		"command":"connect",
		"url":"https://aaa.aaa.aaa"
		"token":"tokenaaaaa"
	}

	OR

	{
		"command":"disconnect"
	}

`
*/

func createWindowsSecurityDescriptor() (sd *windows.SECURITY_DESCRIPTOR, err error) {
	// windows.LookupSID()

	authSID, err := windows.CreateWellKnownSid(windows.WinAuthenticatedUserSid)
	if err != nil {
		return nil, err
	}
	log.Println(utils.FileAndLineStr() + authSID.String())
	// !! DO NOT FREE, created using CreateWellKnownSid
	// defer windows.FreeSid(authSID)
	adminSID, err := windows.CreateWellKnownSid(windows.WinBuiltinAdministratorsSid)
	if err != nil {
		return nil, err
	}
	log.Println(utils.FileAndLineStr() + adminSID.String())
	// !! DO NOT FREE, created using CreateWellKnownSid
	// defer windows.FreeSid(adminSID)
	explicitAcces := []windows.EXPLICIT_ACCESS{
		{
			AccessPermissions: windows.GENERIC_READ | windows.GENERIC_WRITE,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_GROUP,
				TrusteeValue: windows.TrusteeValueFromSID(authSID),
			},
		},
		{
			AccessPermissions: windows.GENERIC_READ | windows.GENERIC_WRITE,
			AccessMode:        windows.GRANT_ACCESS,
			Inheritance:       windows.NO_INHERITANCE,
			Trustee: windows.TRUSTEE{
				TrusteeForm:  windows.TRUSTEE_IS_SID,
				TrusteeType:  windows.TRUSTEE_IS_GROUP,
				TrusteeValue: windows.TrusteeValueFromSID(adminSID),
			},
		},
	}

	acl, err := windows.ACLFromEntries(explicitAcces, nil)
	if err != nil {
		return nil, err
	}

	securityDescriptor, err := windows.NewSecurityDescriptor()
	if err != nil {
		return nil, err
	}

	err = securityDescriptor.SetDACL(acl, true, false)
	if err != nil {
		return nil, err
	}
	return securityDescriptor, nil

}
func manageIncomingData(data []byte, wiredoorPipeHandle windows.Handle) {
	var incomingJson interface{}
	if err := json.Unmarshal(data, &incomingJson); err == nil {
		if jsonObject, ok := incomingJson.(map[string]interface{}); ok {
			if commandStr, ok := jsonObject["command"].(string); ok {
				switch commandStr {
				case "connect":
					url, ok := jsonObject["url"].(string)
					if !ok {
						url = ""
					}
					token, ok := jsonObject["token"].(string)
					if !ok {
						token = ""
					}
					if !wiredoor.WireguardInterfaceExists() {
						wiredoor.Connect(
							wiredoor.ConnectionConfig{
								URL:       url,
								Token:     token,
								UseDaemon: true,
								SetDaemon: false})
					}
					//response
					var writtenLen uint32
					responseData := []byte(`{"response":"ok"}`)
					err := windows.WriteFile(wiredoorPipeHandle, responseData, &writtenLen, nil)
					if err != nil {
						log.Printf(utils.FileAndLineStr()+"error when write to pipe: %v", err)
					}

				case "disconnect":
					wiredoor.Disconnect()
					//response
					var writtenLen uint32
					responseData := []byte(`{"response":"ok"}`)
					err := windows.WriteFile(wiredoorPipeHandle, responseData, &writtenLen, nil)
					if err != nil {
						log.Printf(utils.FileAndLineStr()+"error when write to pipe: %v", err)
					}

				default:
					log.Printf(utils.FileAndLineStr()+"invalid command : %v", commandStr)
				}
			} else {
				log.Printf(utils.FileAndLineStr()+"invalid command type: %v", string(data))
			}
		}

	} else {
		log.Printf(utils.FileAndLineStr()+"error on json decoding `data section(`%s`)` : %v", string(data), err)
	}
}

type wiredoorWindowsService struct{}

func (wsvc *wiredoorWindowsService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {

	log.Printf("Starting service, execute args: %v\n", args)
	s <- svc.Status{State: svc.StartPending}
	//running
	//channel close ends all subroutines
	routineComs := make(chan struct{})
	//wait group for sync
	var waitGroupMonitor sync.WaitGroup

	// create go routine for monitoring

	log.Println("Begin monitoring routine")
	sleepSeconds := serviceInterval
	if sleepSeconds <= 0 {
		sleepSeconds = 10
	}
	//prevent kill when monitoring
	var monitoringMutex sync.Mutex
	go func() {
		for {
			//wait 10 seconds before new check
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
			monitoringMutex.Lock()
			wiredoor.WatchHealt()
			monitoringMutex.Unlock()
		}
	}()

	//routine to manage comunications from no root app

	log.Println("Begin ipc routine")

	//!WARNIG Kill this MF to not block routine

	go func() {

		// log.Printf(utils.FileAndLineStr() + "wiredoorPipeSecurityAttributes")
		sd, err := createWindowsSecurityDescriptor()
		if err != nil {
			log.Printf(utils.FileAndLineStr()+"Error creating windows security descriptor: %v", err)
		}

		wiredoorPipeSecurityAttributes := windows.SecurityAttributes{
			Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
			InheritHandle:      1,
			SecurityDescriptor: sd,
		}
		// log.Printf(utils.FileAndLineStr() + "wiredoorPipeSecurityAttributes done")

		//open server side pipe

		for {
			wiredoorPipeHandle, err := windows.CreateNamedPipe(
				windows.StringToUTF16Ptr(utils.WiredoorPipePathName),
				windows.PIPE_ACCESS_DUPLEX|windows.FILE_FLAG_OVERLAPPED|windows.FILE_FLAG_FIRST_PIPE_INSTANCE,
				windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT|windows.PIPE_REJECT_REMOTE_CLIENTS,
				1,
				1024,
				1024,
				0,
				&wiredoorPipeSecurityAttributes,
				// nil,
			)
			if err != nil {
				log.Printf(utils.FileAndLineStr()+"error creating pipe server on service,%v", err)
				os.Exit(1)
			}
			log.Printf(utils.FileAndLineStr() + "CreateNamedPipe done")

			//wait client
			err = windows.ConnectNamedPipe(wiredoorPipeHandle, nil)
			pipeReady := false
			if err == nil {
				log.Printf(utils.FileAndLineStr() + "Pipe created\n")
				pipeReady = true
			} else {
				if errno, ok := err.(syscall.Errno); ok {
					switch errno {
					case windows.ERROR_PIPE_CONNECTED:
						log.Printf(utils.FileAndLineStr() + "ERROR_PIPE_CONNECTED\n")
						pipeReady = true
					case windows.ERROR_NO_DATA:
						log.Printf(utils.FileAndLineStr()+"ERROR_NO_DATA Pipe closed: %w\n", err)
					case windows.ERROR_PIPE_LISTENING: // not ready, continue
						log.Printf(utils.FileAndLineStr() + "ERROR_PIPE_LISTENING not ready,listening\n")
					case windows.ERROR_PIPE_BUSY:
						log.Println(utils.FileAndLineStr() + "ERROR_PIPE_BUSY")
					case windows.ERROR_INVALID_HANDLE:
						log.Printf(utils.FileAndLineStr()+"ERROR_INVALID_HANDLE invalid server handle: %v", err)
					case windows.ERROR_ACCESS_DENIED:
						log.Printf(utils.FileAndLineStr() + "ERROR_ACCESS_DENIED")
					case windows.ERROR_OPERATION_ABORTED:
						log.Printf(utils.FileAndLineStr() + "ERROR_OPERATION_ABORTED")
					default:
						log.Printf(utils.FileAndLineStr()+"server listen error: %v", err)
					}
				} else {
					log.Printf(utils.FileAndLineStr() + "bad error cast\n")
				}
			}
			// wait incoming data
			log.Printf(utils.FileAndLineStr() + "Start reading")
			if pipeReady {
				var numBytes uint32
				buff := make([]byte, 1024)

				//!TODO move to go routine using overlaped

				err := windows.ReadFile(wiredoorPipeHandle, buff, &numBytes, nil)
				if err == nil {
					//parse
					data := buff[:numBytes]
					manageIncomingData(data, wiredoorPipeHandle)

				} else {
					log.Printf(utils.FileAndLineStr()+"error reading pipe: %v", err)
				}
			} else {
				log.Printf(utils.FileAndLineStr() + "pipe not ready")
			}
			windows.CloseHandle(wiredoorPipeHandle)
		}
	}()
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	log.Printf(utils.FileAndLineStr() + "Start service\n")
	for {
		c := <-r
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending} //notify status
			log.Printf(utils.FileAndLineStr() + "Stop service\n")
			//alert runing goroutine
			close(routineComs)
			//wait for cleanup
			log.Printf(utils.FileAndLineStr() + "Wait for cleanup\n")
			//do not stop monitoring when running
			monitoringMutex.Lock()

			waitGroupMonitor.Wait()
			log.Printf(utils.FileAndLineStr() + "The end\n")
			return false, 0
		default:
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
				log.Println(utils.FileAndLineStr() + "Warinig:Fail to create log file")
			}
			err = svc.Run(utils.WiredoorServiceName, &wiredoorWindowsService{})
			if err != nil {
				log.Print(utils.FileAndLineStr() + "Fail to start service mode\n")
				os.Exit(1)
			}
		} else {
			log.Print(utils.FileAndLineStr() + "Running as console app, made for run as service ...\n")
			os.Exit(1)
		}
		// }
	},
}

func init() {
	rootCmd.AddCommand(windowsServiceCmd)
	windowsServiceCmd.Flags().IntVar(&serviceInterval, "serviceInterval", 10, "Polling interval in seconds (used with service)")
}
