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
	domain      string
	port        int
	path        string
	proto       string
	backendHost string
	allowList   []string
	blockList   []string
	ttl         string
)

var httpCmd = &cobra.Command{
	Use:   "http <name>",
	Short: "Expose a local HTTP service via Wiredoor",
	Long: `Expose a local HTTP service using the Wiredoor reverse proxy.

This command allows you to publish a service running on your local machine or private network
by assigning it a public domain and path. It creates a secure HTTPS route automatically
(using Let's Encrypt or self-signed certificates depending on DNS resolution).

Required arguments:
  <name>           Unique name for the exposed service (used for management)

Required flags:
  --domain         Public domain to expose the service under
  --port           Local port where your service is running

Optional flags:
  --proto          Protocol to use for local service ("http" or "https", defaults to "http")
  --backendHost    Useful when the node is the gateway and needs to forward to another internal host (defaults to "localhost")
  --path           URL path to expose (defaults to "/")
  --allow          Comma-separated list of allowed IP addresses or CIDRs (access control)
  --block          Comma-separated list of blocked IP addresses or CIDRs (access control)
	--ttl						 Time-to-live duration for the exposure (e.g., "30m", "1h", "2d").
                   Automatically disables the service after the specified duration.

Example scenario:
  You have a local service running on http://localhost:3000 and want to expose it as:
  https://website.com/ui

  This command will handle certificate provisioning and route configuration:
    wiredoor http my-website --domain website.com --port 3000 --path /ui

  You can also restrict access to specific IPs:
    wiredoor http my-website --domain website.com --port 3000 --allow 192.168.1.0/24

Certificates:
  - If the domain is public and resolves to the Wiredoor gateway, a valid certificate is obtained via Let's Encrypt.
  - If the domain is private or doesn't resolve, a self-signed certificate will be used.`,
	Example: `  # Basic HTTP exposure
  wiredoor http my-website --domain website.com --port 3000

  # Use a custom path and HTTPS
  wiredoor http my-website --domain website.com --port 3000 --path /ui --proto https

  # Forward to another backend host (Only if your configured node is a Gateway)
  wiredoor http my-website --domain website.com --proto https --backendHost 10.0.0.100 --port 443

  # Restrict access to specific IP ranges
  wiredoor http my-website --domain website.com --port 3000 --allow 192.168.1.0/24

  # Block a specific IP
  wiredoor http my-website --domain website.com --port 3000 --block 203.0.113.42
	
	# Expose service temporarily for 1 hour
  wiredoor http my-website --domain website.com --port 3000 --ttl 1h`,
	Args: cobra.ExactArgs(1), // require "name"
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]

		node := wiredoor.GetNode()

		if node.IsGateway && backendHost == "" {
			fmt.Println("You must define --backendHost when your node is a gateway")
			return
		}

		wiredoor.ExposeHTTP(wiredoor.HttpServiceParams{
			Name:         name,
			Domain:       domain,
			BackendPort:  port,
			BackendProto: proto,
			BackendHost:  backendHost,
			PathLocation: path,
			AllowedIps:   allowList,
			BlockedIps:   blockList,
			Ttl:          ttl,
		}, node)
	},
}

func init() {
	rootCmd.AddCommand(httpCmd)

	httpCmd.Flags().StringVar(&domain, "domain", "", "Public domain to expose the service under (required)")
	httpCmd.Flags().IntVar(&port, "port", 0, "Local port where your HTTP service is running (required)")
	httpCmd.Flags().StringVar(&path, "path", "/", "URL path to expose (default: \"/\")")
	httpCmd.Flags().StringVar(&proto, "proto", "http", "Protocol to expose the service as (http or https)")
	httpCmd.Flags().StringVar(&backendHost, "backendHost", "", "Target internal IP or hostname (used by gateway nodes)")
	httpCmd.Flags().StringSliceVar(&allowList, "allow", nil, "List of allowed IPs or CIDRs")
	httpCmd.Flags().StringSliceVar(&blockList, "block", nil, "List of blocked IPs or CIDRs")
	httpCmd.Flags().StringVar(&ttl, "ttl", "", "Time-to-live duration for the exposure (e.g., 30m, 1h, 2d)")
}
