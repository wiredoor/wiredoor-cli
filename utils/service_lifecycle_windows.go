//go:build windows

package utils

import (
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceTimeout = 30 * time.Second

// managedService keeps state transitions testable without changing real services.
type managedService interface {
	Query() (svc.Status, error)
	Control(svc.Cmd) (svc.Status, error)
	Start(...string) error
	Config() (mgr.Config, error)
	UpdateConfig(mgr.Config) error
	Delete() error
	Close() error
}

func waitForServiceState(s managedService, target svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("query service: %w", err)
		}
		if status.State == target {
			return nil
		}
		if target == svc.Running && status.State == svc.Stopped {
			return fmt.Errorf("service stopped before reaching Running (exit code %d)", status.Win32ExitCode)
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("timed out waiting for service state %d (current state %d)", target, status.State)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func stopManagedService(s managedService) error {
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("query service: %w", err)
	}
	switch status.State {
	case svc.Stopped:
		return nil
	case svc.StopPending:
		return waitForServiceState(s, svc.Stopped, serviceTimeout)
	case svc.StartPending, svc.ContinuePending:
		if err := waitForServiceState(s, svc.Running, serviceTimeout); err != nil {
			// Startup may have failed while we were waiting; it is then already stopped.
			if status, queryErr := s.Query(); queryErr == nil && status.State == svc.Stopped {
				return nil
			}
			return err
		}
	case svc.PausePending:
		if err := waitForServiceState(s, svc.Paused, serviceTimeout); err != nil {
			return err
		}
	}
	if _, err := s.Control(svc.Stop); err != nil && !errors.Is(err, windows.ERROR_SERVICE_NOT_ACTIVE) {
		return fmt.Errorf("stop service: %w", err)
	}
	return waitForServiceState(s, svc.Stopped, serviceTimeout)
}

func deleteManagedService(s managedService) error {
	defer s.Close()
	if err := stopManagedService(s); err != nil {
		return err
	}
	if err := s.Delete(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		return fmt.Errorf("delete service: %w", err)
	}
	return nil
}

func startManagedService(s managedService) error {
	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("query service: %w", err)
	}
	if status.State == svc.Running {
		return nil
	}
	if status.State == svc.StartPending {
		return waitForServiceState(s, svc.Running, serviceTimeout)
	}
	if err := stopManagedService(s); err != nil {
		return err
	}
	if err := s.Start(); err != nil && !errors.Is(err, windows.ERROR_SERVICE_ALREADY_RUNNING) {
		return fmt.Errorf("start service: %w", err)
	}
	return waitForServiceState(s, svc.Running, serviceTimeout)
}

func installExistingService(s managedService, executable string) error {
	if err := stopManagedService(s); err != nil {
		return err
	}
	config, err := s.Config()
	if err != nil {
		return fmt.Errorf("read service configuration: %w", err)
	}
	config.BinaryPathName = fmt.Sprintf(`"%s" service --serviceInterval 10`, executable)
	config.StartType = mgr.StartAutomatic
	config.ServiceType = windows.SERVICE_WIN32_OWN_PROCESS
	config.ServiceStartName = "LocalSystem"
	if err := s.UpdateConfig(config); err != nil {
		return fmt.Errorf("update service: %w", err)
	}
	return startManagedService(s)
}

// InstallService creates or updates the service and waits until it is running.
func InstallService() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return fmt.Errorf("connect to service manager: %w", err)
	}
	defer m.Disconnect()
	s, err := m.OpenService(WiredoorServiceName)
	if errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
		if err := waitForServiceDeletion(m, WiredoorServiceName); err != nil {
			return err
		}
		s, err = m.OpenService(WiredoorServiceName)
	}
	if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
		s, err = m.CreateService(WiredoorServiceName, executable, mgr.Config{
			DisplayName: WiredoorServiceName,
			Description: "Wiredoor Service",
			StartType:   mgr.StartAutomatic,
			ServiceType: windows.SERVICE_WIN32_OWN_PROCESS,
		}, "service", "--serviceInterval", "10")
		if err != nil {
			return fmt.Errorf("create service: %w", err)
		}
		defer s.Close()
		return startManagedService(s)
	}
	if err != nil {
		return fmt.Errorf("open service: %w", err)
	}
	defer s.Close()
	return installExistingService(s, executable)
}

func waitForServiceDeletion(m *mgr.Mgr, name string) error {
	deadline := time.Now().Add(serviceTimeout)
	for {
		s, err := m.OpenService(name)
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return nil
		}
		if err != nil && !errors.Is(err, windows.ERROR_SERVICE_MARKED_FOR_DELETE) {
			return fmt.Errorf("check service deletion: %w", err)
		}
		if s != nil {
			s.Close()
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("timed out waiting for service deletion")
		}
		time.Sleep(100 * time.Millisecond)
	}
}
