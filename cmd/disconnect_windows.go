//go:build windows
// +build windows

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect this node from the Wiredoor server",
	Long: `Gracefully disconnect this node from the Wiredoor server and stop the VPN tunnel.

This command tears down the active WireGuard connection and disables access to all exposed services.
No configuration is deleted, so you can reconnect later using 'wiredoor connect'.

Typical use cases:
  - Temporarily stopping the connection
  - Restarting the tunnel
  - Preparing the node for maintenance

Note:
  This does NOT delete the node or token from the Wiredoor server. Use 'wiredoor disable' if you only want to stop a specific service.

Examples:
  wiredoor disconnect
  wiredoor disconnect && sleep 5 && wiredoor connect`,
	Run: func(cmd *cobra.Command, args []string) {

		isWindowsService, err := svc.IsWindowsService()
		if err != nil {

			log.Print(utils.FileAndLineStr()+"error detecting if I am a service, %v\n", err)
			os.Exit(1)
		}
		if isWindowsService {
			log.Print(utils.FileAndLineStr() + "error, connect command not usable as service")
			os.Exit(1)
		}

		//1 connect to pipe to read/write non blocking capable
		wiredoorPipeHandle, err := windows.CreateFile(
			windows.StringToUTF16Ptr(utils.WiredoorPipePathName),
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_OVERLAPPED, //overlaped read
			0)

		if err != nil {
			log.Printf(utils.FileAndLineStr()+"error opening service pipe, is service running? : %v", err)
			os.Exit(1)
		}
		defer windows.CloseHandle(wiredoorPipeHandle)
		//2 send disconnect message

		//prepare data to send:
		jsonToSend := make(map[string]interface{})
		jsonToSend["command"] = "disconnect"

		if data, err := json.Marshal(jsonToSend); err == nil {
			var writtenLen uint32
			err := windows.WriteFile(wiredoorPipeHandle, data, &writtenLen, nil)
			if err != nil {
				log.Printf(utils.FileAndLineStr()+"error when write to pipe: %v", err)
				os.Exit(1)
			}
			if int(writtenLen) != len(data) {
				log.Printf(utils.FileAndLineStr()+"Warninig message not fully sended, sended %v of %v bytes", writtenLen, len(data))
			}
		} else {
			log.Printf(utils.FileAndLineStr()+"Marshal error :%v\n", err)
			os.Exit(1)
		}
		//3 wait for response or termination for check status
		readChanErr := make(chan error, 1)
		var readLen uint32
		readBuff := make([]byte, 1024)
		// blocking operation to another thread, using events and overlaped to avoid hang for ever
		go func() {
			defer close(readChanErr)
			if readEvent, err := windows.CreateEvent(nil, 1, 0, nil); err == nil {
				defer windows.CloseHandle(readEvent)

				readOverlaped := new(windows.Overlapped)
				readOverlaped.HEvent = readEvent

				err := windows.ReadFile(wiredoorPipeHandle, readBuff, &readLen, readOverlaped)
				if err != nil && err != windows.ERROR_IO_PENDING {
					log.Printf(utils.FileAndLineStr()+"overlaped read error: %v", err)
					os.Exit(1)
				}
				// 10 seconds = 10000 miliseconds
				readStatus, err := windows.WaitForSingleObject(readOverlaped.HEvent, uint32(10000))
				if err != nil {
					log.Printf(utils.FileAndLineStr()+"event wait error: %v", err)
					os.Exit(1)
				}
				switch readStatus {
				case windows.WAIT_OBJECT_0:
					//HURRA
					err = windows.GetOverlappedResult(wiredoorPipeHandle, readOverlaped, &readLen, true)
					if err != nil {
						log.Printf(utils.FileAndLineStr()+"err on get overlaped result: %v", err)
						os.Exit(1)
					}
				case uint32(windows.WAIT_TIMEOUT):
					readChanErr <- fmt.Errorf("WAIT_TIMEOUT")
				}
				readChanErr <- err
			} else {
				log.Printf(utils.FileAndLineStr()+"event creation error: %v", err)
				os.Exit(1)
			}
			readChanErr <- nil
		}()
		select {
		case <-time.After(10 * time.Second):
			log.Printf(utils.FileAndLineStr() + "Warinig, service response timed out after 10 seconds")
		case err, ok := <-readChanErr:
			if !ok {
				log.Printf(utils.FileAndLineStr() + "I/O error,read channel closed")
				os.Exit(1)
			}
			if err != nil {
				log.Printf(utils.FileAndLineStr()+"read pipe error: %v", err)
				os.Exit(1)
			}
			data := readBuff[:readLen]
			jsonResponse := make(map[string]interface{})
			if err := json.Unmarshal(data, &jsonResponse); err == nil {
				if response, ok := jsonResponse["response"].(string); ok {
					switch response {
					case "ok":
						wiredoor.Status()
						os.Exit(0)
					default:
						log.Printf(utils.FileAndLineStr()+"unhandled service reposnse: %v", response)
						os.Exit(1)
					}

				} else {
					log.Printf(utils.FileAndLineStr()+"response format error: %v", data)
				}
			} else {
				log.Printf(utils.FileAndLineStr()+"json response error: %v", err)
				os.Exit(1)
			}

		}
		//------------------------------------------------

		//!TODO move to use service

		//check for admin
		// if !utils.IsRoot() {
		// 	//run as admin on windows or print a message on linux
		// 	if err := utils.RelaunchAsRoot(); err == nil {
		// 		// wiredoor.Status()
		// 		os.Exit(0)
		// 	}
		// }
		//! TODO Continue anyway ?
		// wiredoor.Disconnect()
	},
}

func init() {
	rootCmd.AddCommand(disconnectCmd)
}
