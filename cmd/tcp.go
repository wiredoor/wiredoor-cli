/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var (
	tcpPort        int
	tcpDomain      string
	tcpSSL         bool
	tcpProto       string
	tcpBackendHost string
	tcpAllowList   []string
	tcpBlockList   []string
)

var tcpCmd = &cobra.Command{
	Use:   "tcp <name>",
	Short: "Expose a local TCP or UDP service via Wiredoor",
	Long: `Expose a generic TCP or UDP service to the public using the Wiredoor gateway.

This command allows you to publish services like SSH, PostgreSQL, Redis, or custom protocols
by assigning a public port on the Wiredoor gateway and forwarding traffic to your internal service.

Required arguments:
  <name>           Unique name for the exposed service (used for management)

Required flags:
  --port           Local backend port of the service

Optional flags:
  --domain         Optional domain label (used for organization only; TCP services are accessed via port)
  --ssl            Enable SSL termination (wrap raw TCP stream in TLS)
  --proto          Backend protocol: "tcp" or "udp" (default: tcp)
  --backendHost    Host IP or name (used when the node is a gateway proxying to internal backends)
  --allowedIps     List of allowed source IPs or CIDRs
  --blockedIps     List of blocked source IPs or CIDRs

How it works:
  - Wiredoor assigns a public port (e.g. 20000) on the gateway.
  - Incoming traffic to this port is forwarded to your service running on the local/private network.

Examples:
  # Expose local service running on port 22 (SSH)
  wiredoor tcp ssh-access --port 22

  # Expose Redis with SSL and access control
  wiredoor tcp redis --port 6379 --proto tcp --ssl --allowedIps 10.0.0.0/8

  # Expose UDP service (e.g., DNS)
  wiredoor tcp dns --port 53 --proto udp

  # Expose a PostgreSQL instance via a gateway node
  wiredoor tcp db --port 5432 --backendHost 10.0.0.100`,
	Example: `  # Simple TCP exposure
  wiredoor tcp ssh-access --port 22

  # TCP with SSL enabled
  wiredoor tcp redis --port 6379 --ssl

  # UDP service
  wiredoor tcp telemetry --port 9999 --proto udp

  # Gateway mode with access restriction
  wiredoor tcp db --port 5432 --backendHost 10.1.0.15 --allowedIps 192.168.1.0/24`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		node := wiredoor.GetNode()

		if node.IsGateway && backendHost == "" {
			fmt.Println("You must define --backendHost when your node is a gateway")
			return
		}

		wiredoor.ExposeTCP(wiredoor.TcpServiceParams{
			Name:        name,
			Domain:      tcpDomain,
			Proto:       tcpProto,
			BackendPort: tcpPort,
			BackendHost: tcpBackendHost,
			AllowedIps:  tcpAllowList,
			BlockedIps:  tcpBlockList,
			Ssl:         tcpSSL,
		}, node)
	},
}

func init() {
	rootCmd.AddCommand(tcpCmd)

	tcpCmd.Flags().IntVar(&tcpPort, "port", 0, "Backend port of the local service (required)")
	tcpCmd.Flags().StringVar(&tcpProto, "proto", "tcp", "Protocol used by the backend service (tcp or udp)")
	tcpCmd.Flags().BoolVar(&tcpSSL, "ssl", false, "Wrap TCP connection in SSL (TLS)")
	tcpCmd.Flags().StringVar(&tcpDomain, "domain", "", "Optional domain label (for organizational purposes)")
	tcpCmd.Flags().StringVar(&tcpBackendHost, "backendHost", "", "Backend host IP or hostname (used by gateway nodes)")
	tcpCmd.Flags().StringSliceVar(&tcpAllowList, "allowedIps", nil, "List of allowed IPs or CIDRs")
	tcpCmd.Flags().StringSliceVar(&tcpBlockList, "blockedIps", nil, "List of blocked IPs or CIDRs")
}
