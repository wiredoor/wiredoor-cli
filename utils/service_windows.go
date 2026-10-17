//go:build windows
// +build windows

package utils

import (
	"errors"
	"fmt"
	"log/slog"
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
// DeleteService stops and removes a service. An absent service is already uninstalled.
func DeleteService(serviceName string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return nil
	}
	if errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return waitForServiceDeletion(m, serviceName)
	}
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	if err := deleteManagedService(s); err != nil {
		return err
	}
	// The handle is closed before waiting for Windows to finish deletion.
	return waitForServiceDeletion(m, serviceName)
}
func StartService(serviceName string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	defer s.Close()
	config, err := s.Config()
	if err != nil {
		return fmt.Errorf("read service configuration: %w", err)
	}
	if config.StartType != mgr.StartAutomatic {
		config.StartType = mgr.StartAutomatic
		if err := s.UpdateConfig(config); err != nil {
			return fmt.Errorf("update service: %w", err)
		}
	}
	return startManagedService(s)
}

// StopService succeeds if the service is absent or already stopped.
func StopService(serviceName string) error {
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		return nil
	}
	if errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return waitForServiceDeletion(m, serviceName)
	}
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	defer s.Close()
	return stopManagedService(s)
}

// RestartService starts an installed service after waiting for it to stop.
func RestartService() error {
	if err := StopService(WiredoorServiceName); err != nil {
		return err
	}
	return StartService(WiredoorServiceName)
}

func EnableService() error {
	exists, err := WiredoorServiceExists()
	if err != nil {
		return err
	}
	if !exists {
		return InstallService()
	}
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
