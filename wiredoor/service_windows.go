//go:build windows
// +build windows

package wiredoor

import (
	"fmt"
	"os"
	"slices"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
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

var WiredoorServiceName = "wiredoorService"
var serviceArgs = "status --watch --interval 10"

// does not needs Admin priv. on windows
func WiredoorServiceExists() (bool, error) {
	return serviceExists(WiredoorServiceName)
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

func WiredoorServiceRunning() (bool, error) {
	if exists, err := WiredoorServiceExists(); err == nil {
		if exists {
			return ServiceRunning(WiredoorServiceName)
		} else {
			return false, fmt.Errorf("service does not exists")
		}
	} else {
		return false, fmt.Errorf("could not determine if service exists: %v", err)
	}

}
func ServiceRunning(serviceName string) (bool, error) {
	// open service manager connection as common user
	serviceManagerConnection, err := connectToLocalServiceManagerLowPriv()
	if err != nil {
		return false, fmt.Errorf("unable to connect service manager: %v", err)
	}
	defer serviceManagerConnection.Disconnect()

	// check service if exists
	if serviceList, err := serviceManagerConnection.ListServices(); err == nil {
		if slices.Contains(serviceList, serviceName) {
			if serviceX, err := openServiceLowPriv(serviceManagerConnection, serviceName); err == nil {
				if status, err := serviceX.Query(); err == nil {
					return status.State == svc.Running, nil
				} else {
					return false, fmt.Errorf("unable to query service status: %v", err)
				}
			} else {
				return false, fmt.Errorf("unable to open service: %v", err)
			}

		} else {
			return false, fmt.Errorf("service not found")
		}
	} else {
		return false, fmt.Errorf("unable to list services: %v", err)
	}
}

// --------------------------------------------------------------
// Common api
// needs Admin priv
// StartService starts the wiredoor service
func StartService() error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, WiredoorServiceName) {
				//create service
				appPath, err := os.Executable()
				if err != nil {
					return fmt.Errorf("unable to find service path: %v", err)
				}
				serviceConfig := mgr.Config{
					DisplayName:    WiredoorServiceName,
					Description:    "Wiredoor service",
					BinaryPathName: appPath,
					StartType:      mgr.StartManual,
					ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
				}
				serviceCon, err := serviceMangerConnection.CreateService(WiredoorServiceName, appPath, serviceConfig, serviceArgs)
				if err != nil {
					return fmt.Errorf("error creating service: %v", err)
				}
				defer serviceCon.Close()
			}
			//start service
			if serviceConnection, err := serviceMangerConnection.OpenService(WiredoorServiceName); err == nil {
				//!TODO check if serviceArgs are pased as command line or as service args (not the same)
				defer serviceConnection.Close()
				return fmt.Errorf("unable to start service: %v", serviceConnection.Start(serviceArgs))
			} else {
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}

// StopService stops the wiredoor service
func StopService() error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, WiredoorServiceName) {
				//not listed
				return fmt.Errorf("not stoped, service not found: %v", err)
			}
			//start service
			if serviceConnection, err := serviceMangerConnection.OpenService(WiredoorServiceName); err == nil {
				//!TODO check if serviceArgs are pased as command line or as service args (not the same)
				if _, err := serviceConnection.Control(svc.Stop); err != nil {
					return fmt.Errorf("unable to stop service: %v", err)
				}
				return nil
			} else {
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}

// RestartService restarts the wiredoor service
func RestartService() error {
	err := StopService()
	if err == nil {
		err = StartService()
		return err
	}
	return nil
}

func EnableService() error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, WiredoorServiceName) {
				//not listed
				return fmt.Errorf("not disabled, service not found: %v", err)
			}
			//start service
			if serviceConnection, err := serviceMangerConnection.OpenService(WiredoorServiceName); err == nil {
				defer serviceConnection.Close()
				//!TODO check if serviceArgs are pased as command line or as service args (not the same)
				if serviceCfg, err := serviceConnection.Config(); err != nil {
					return fmt.Errorf("unable to get service config: %v", err)
				} else {
					serviceCfg.StartType = mgr.StartAutomatic
					if err := serviceConnection.UpdateConfig(serviceCfg); err != nil {
						return fmt.Errorf("unable to update service config: %v", err)
					}
					StartService()
				}
				return nil
			} else {
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}

// DisableService disables the wiredoor service from starting on boot
func DisableService() error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, WiredoorServiceName) {
				//not listed
				return fmt.Errorf("not disabled, service not found: %v", err)
			}
			//start service
			if serviceConnection, err := serviceMangerConnection.OpenService(WiredoorServiceName); err == nil {
				defer serviceConnection.Close()
				//!TODO check if serviceArgs are pased as command line or as service args (not the same)
				if serviceCfg, err := serviceConnection.Config(); err != nil {
					return fmt.Errorf("unable to get service config: %v", err)
				} else {
					serviceCfg.StartType = mgr.StartDisabled
					if err := serviceConnection.UpdateConfig(serviceCfg); err != nil {
						return fmt.Errorf("unable to update service config: %v", err)
					}
					StopService()
				}
				return nil
			} else {
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}
