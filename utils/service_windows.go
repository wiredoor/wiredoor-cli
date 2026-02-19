//go:build windows
// +build windows

package utils

import (
	"fmt"
	"log/slog"
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

// var serviceArgs = "service --serviceInterval 10"

// does not needs Admin priv. on windows
func WiredoorServiceExists() (bool, error) {
	return ServiceExists(WiredoorServiceName)
}
func ServiceExists(serviceName string) (bool, error) {
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
			return false, nil
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
				defer serviceX.Close()
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
func DeleteService(serviceName string) error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
	} else {
		return fmt.Errorf("unable to connect to service manager: %v", err)
	}
	if serviceConnection, err := serviceMangerConnection.OpenService(serviceName); err == nil {
		defer serviceConnection.Close()
		if err := serviceConnection.Delete(); err != nil {
			slog.Error("unable to delete service", "error", err)
			return fmt.Errorf("unable to delete service: %v", err)
		}
		return nil
	} else {
		slog.Error("unable to open service", "error", err)
		return fmt.Errorf("unable to open service: %v", err)
	}
}
func CreateServiceFromThisExecutable(serviceName, user, passwd string) error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		appPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("unable to find service executable path: %v", err)
		}
		var serviceConfig mgr.Config
		if len(user) <= 0 || len(passwd) <= 0 {
			//WARNING RUN AS SYSTEM
			serviceConfig = mgr.Config{
				DisplayName: serviceName,
				Description: "Wiredoor Service",
				// BinaryPathName: appPath + " " + serviceArgs,
				BinaryPathName: fmt.Sprintf(`"%s"`, appPath),
				StartType:      mgr.StartAutomatic,
				ServiceType:    windows.SERVICE_WIN32_OWN_PROCESS,
			}
		} else {
			fmt.Printf("user : %s\t|\tpasswd : %s\n", user, passwd)
			serviceConfig = mgr.Config{
				DisplayName: serviceName,
				Description: "Wiredoor Service",
				// BinaryPathName: appPath + " " + serviceArgs,
				BinaryPathName:   fmt.Sprintf(`"%s"`, appPath),
				StartType:        mgr.StartAutomatic,
				ServiceType:      windows.SERVICE_WIN32_OWN_PROCESS,
				ServiceStartName: user,
				Password:         passwd,
			}
		}
		// time.Sleep(3 * time.Second)
		serviceCon, err := serviceMangerConnection.CreateService(serviceName, appPath, serviceConfig, "service", "--serviceInterval", "10")
		if err != nil {
			return fmt.Errorf("error creating service: %v", err)
		}
		defer serviceCon.Close()
		return nil
	} else {
		return fmt.Errorf("unable to connect to service manager: %v", err)
	}
}
func StartService(serviceName string) error {
	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
	} else {
		return fmt.Errorf("unable to connect to service manager: %v", err)
	}
	if serviceConnection, err := serviceMangerConnection.OpenService(serviceName); err == nil {
		//serviceArgs are pased as service args plus app args on BinaryPathName field of serviceConfig
		defer serviceConnection.Close()
		if serviceCfg, err := serviceConnection.Config(); err != nil {
			slog.Error("unable to get service config", "error", err)
		} else {
			serviceCfg.StartType = mgr.StartAutomatic
			if err := serviceConnection.UpdateConfig(serviceCfg); err != nil {
				slog.Error("unable to update service config", "error", err)
			}
		}
		if err := serviceConnection.Start(""); err != nil {
			slog.Error("unable to start service", "error", err)
			return fmt.Errorf("unable to start service: %v", err)
		}
		return nil
	} else {
		slog.Error("unable to open service", "error", err)
		return fmt.Errorf("unable to open service: %v", err)
	}
}

// StopService stops the wiredoor service
func StopService(serviceName string) error {

	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, serviceName) {
				//not listed
				slog.Error("not stoped, wiredoor service not found")
				return fmt.Errorf("not stoped, wiredoor service not found")
			}
			//query stop service
			if serviceConnection, err := serviceMangerConnection.OpenService(serviceName); err == nil {
				if _, err := serviceConnection.Control(svc.Stop); err != nil {
					slog.Error("unable to stop service", "error", err)
					return fmt.Errorf("unable to stop service: %v", err)
				}
				return nil
			} else {
				slog.Error("unable to open service", "error", err)
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			slog.Error("error listing serices", "error", err)
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		slog.Error("unable to access to service manager", "error", err)
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}

// RestartService restarts the wiredoor service
func RestartService() error {
	if exists, _ := WiredoorServiceExists(); exists {
		err := StopService(WiredoorServiceName)
		if err != nil {
			slog.Warn("When try to stop service", "error", err)
		}
	} else {
		err := CreateServiceFromThisExecutable(WiredoorServiceName, "", "")
		if err != nil {
			return err
		}
	}
	return StartService(WiredoorServiceName)
}

func EnableService() error {
	return RestartService()
}

// DisableService disable service from starting on boot
func DisableService(serviceName string) error {

	serviceMangerConnection, err := mgr.Connect()
	if err == nil {
		defer serviceMangerConnection.Disconnect()
		// detect if installed
		if serviceList, err := serviceMangerConnection.ListServices(); err == nil {
			if !slices.Contains(serviceList, serviceName) {
				//not listed
				// return fmt.Errorf("not disabled, service not found: %v", err)
				return nil
			}
			//disable service
			if serviceConnection, err := serviceMangerConnection.OpenService(serviceName); err == nil {
				defer serviceConnection.Close()
				if serviceCfg, err := serviceConnection.Config(); err != nil {
					slog.Error("unable to get service config", "error", err)
					return fmt.Errorf("unable to get service config: %v", err)
				} else {
					serviceCfg.StartType = mgr.StartDisabled
					if err := serviceConnection.UpdateConfig(serviceCfg); err != nil {
						slog.Error("unable to update service config", "error", err)
						return fmt.Errorf("unable to update service config: %v", err)
					}
					StopService(serviceName)
				}
				return nil
			} else {
				slog.Error("unable to open service", "error", err)
				return fmt.Errorf("unable to open service: %v", err)
			}
		} else {
			slog.Error("error listing serices", "error", err)
			return fmt.Errorf("error listing serices: %v", err)
		}
	} else {
		slog.Error("unable to access to service manager", "error", err)
		return fmt.Errorf("unable to access to service manager: %v", err)
	}
}
