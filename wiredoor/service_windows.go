//go:build windows
// +build windows

package wiredoor

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc/mgr"
)

// ---------------------------------------
// not admind variant of service manager functionality
func connectToLocalServiceManagerLowPriv() (*mgr.Mgr, error) {
	var s *uint16
	h, err := windows.OpenSCManager(s, nil, windows.SC_MANAGER_CONNECT|windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return nil, err
	}
	return &mgr.Mgr{Handle: h}, nil
}

func openServiceLowPriv(m *mgr.Mgr, name string) (*mgr.Service, error) {
	h, err := windows.OpenService(m.Handle, syscall.StringToUTF16Ptr(name), windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return nil, err
	}
	return &mgr.Service{Name: name, Handle: h}, nil
}

//---------------------------------------

var WiredorServiceName = "wiredoorService"

// does not needs Admin priv. on windows
func WiredoorServiceExists() (bool, error) {
	return serviceExists(WiredorServiceName)
}
func serviceExists(serviceName string) (bool, error) {
	// Connect to service manager
	serviceManagerConnection, err := connectToLocalServiceManagerLowPriv()
	if err != nil {
		return false, fmt.Errorf("unable to connect service manager: %v", err)
	}
	defer serviceManagerConnection.Disconnect()

	// List services and check
	if serviceList, err := serviceManagerConnection.ListServices(); err == nil {
		return slices.Contains(serviceList, serviceName), nil
	} else {
		return false, fmt.Errorf("unable to list services: %v", err)
	}
}

//!TODO continue
/*
serviceManagerConnection, err := connectToLocalServiceManagerLowPriv()
	if err != nil {
		fmt.Printf("unable to connect service manager: %v\n", err)
		os.Exit(3)
	}
	defer serviceManagerConnection.Disconnect()

	// List services and check
	if serviceList, err := serviceManagerConnection.ListServices(); err == nil {
		for _,serviceIName:= range serviceList{
			fmt.Printf("Service name: %v : ",serviceIName)
				if serviceX,err := openServiceLowPriv(serviceManagerConnection,serviceIName); err == nil{
					if status,err := serviceX.Query(); err == nil{
						fmt.Printf("%v\n",StateString(status.State))
					}else{
						fmt.Printf("unable to query service status: %v\n", err)
					}
				}else{
					fmt.Printf("unable to open service: %v\n", err)
				}
		}
	}else{
		fmt.Printf("unable to connect service manager: %v", err)
		os.Exit(3)
	}

*/
func WiredoorServiceRunning() (bool, error) {
	return true, nil
}
func ServiceRunning() (bool, error) {
	return true, nil
}

// StartService starts the wiredoor service based on init system
func StartService() error {
	return fmt.Errorf("unsupported service")
}

// StopService stops the wiredoor service
func StopService() error {
	return fmt.Errorf("unsupported service")
}

// RestartService restarts the wiredoor service
func RestartService() error {
	return fmt.Errorf("unsupported service")
}

func EnableService() error {
	//!TODO integrate all systems
	return fmt.Errorf("unsupported service")
}

// DisableService disables the wiredoor service from starting on boot
func DisableService() error {
	//!TODO integrate all systems
	return fmt.Errorf("unsupported service")
}

// run executes a command and prints output
func run(cmd []string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
