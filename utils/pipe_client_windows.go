//go:build windows
// +build windows

package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"golang.org/x/sys/windows"
)

/*
Copyright © 2024 Daniel Mesa <support@wiredoor.net>
*/
var WiredoorPipePathName string = `\\.\pipe\wiredoorServicePipe`

//Example data to send:
// jsonToSend := make(map[string]interface{})
// jsonToSend["command"] = "connect"
// jsonToSend["url"] = url
// jsonToSend["token"] = token
// jsonToSend["daemos"] = useDaemon

// return a [] byte from remote service, or/and a textual error

func ExecuteLocalSystemServiceTask(jsonToSend map[string]interface{}) ([]byte, error) {
	//1 connect to pipe
	wiredoorPipeHandle, err := windows.CreateFile(
		windows.StringToUTF16Ptr(WiredoorPipePathName),
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		0,
		nil,
		windows.OPEN_EXISTING,
		windows.FILE_FLAG_OVERLAPPED, //overlaped read
		0)

	if err != nil {
		return nil, fmt.Errorf("error opening service pipe, is service running? : %v", err)
	}
	defer windows.CloseHandle(wiredoorPipeHandle)
	//2 send connect message

	if data, err := json.Marshal(jsonToSend); err == nil {
		var writtenLen uint32
		err := windows.WriteFile(wiredoorPipeHandle, data, &writtenLen, nil)
		if err != nil {
			return nil, fmt.Errorf("error when write to pipe: %v", err)
		}
		if int(writtenLen) != len(data) {
			log.Printf("Warninig message not fully sended, sended %v of %v bytes", writtenLen, len(data))
		}
	} else {
		return nil, fmt.Errorf("marshal error: %v", err)
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
				log.Printf("overlaped read error: %v", err)
				return
			}
			// 10 seconds = 10000 miliseconds
			readStatus, err := windows.WaitForSingleObject(readOverlaped.HEvent, uint32(10000))
			if err != nil {
				log.Printf("event wait error: %v", err)
				return
			}
			switch readStatus {
			case windows.WAIT_OBJECT_0:
				//HURRA
				err = windows.GetOverlappedResult(wiredoorPipeHandle, readOverlaped, &readLen, true)
				if err != nil {
					log.Printf("err on get overlaped result: %v", err)
					return
				}
			case uint32(windows.WAIT_TIMEOUT):
				readChanErr <- fmt.Errorf("WAIT_TIMEOUT")
			}
			readChanErr <- err
		} else {
			log.Printf("event creation error: %v", err)
			return
		}
		readChanErr <- nil
	}()
	//end ruitine
	select {
	case <-time.After(10 * time.Second):
		log.Printf("Warinig, service response timed out after 10 seconds")
	case err, ok := <-readChanErr:
		if !ok {
			return nil, fmt.Errorf("I/O error,read channel closed")
		}
		if err != nil {
			return nil, fmt.Errorf("read pipe error: %v", err)
		}
		data := readBuff[:readLen]
		return data, nil

	}

	return nil, nil
}
