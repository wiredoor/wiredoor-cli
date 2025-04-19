package utils

import (
	"fmt"
	"net"
	"time"
)

func CheckPort(host string, port int) bool {
	address := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	timeout := 2 * time.Second

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
			fmt.Printf("Port %d is closed or unreachable: %v\n", port, err)
			return false
	}
	defer conn.Close()

	return true
}

func LocalTunnelIP() string {
	iface, err := net.InterfaceByName("wg0")
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil || len(addrs) == 0 {
		return ""
	}
	ip, _, _ := net.ParseCIDR(addrs[0].String())
	return ip.String()
}

func LocalServerIP() string {
	iface, err := net.InterfaceByName("wg0")
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil || len(addrs) == 0 {
		return ""
	}
	ip, _, _ := net.ParseCIDR(addrs[0].String())

	ipv4 := ip.To4()

	serverIP := make(net.IP, len(ipv4))
	copy(serverIP, ipv4)

	serverIP[3] = 1

	return serverIP.String()
}

func DefaultSubnet() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// ignore inactive interfaces or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}

			if ipNet == nil || ipNet.IP.To4() == nil {
				continue
			}

			return ipNet.String(), nil // IP + mask
		}
	}

	return "", fmt.Errorf("unable to get network information")
}
