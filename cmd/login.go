/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"
	"github.com/wiredoor/wiredoor-cli/utils"
	"github.com/wiredoor/wiredoor-cli/wiredoor"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Wiredoor server and register this node",
	Long: `Authenticate to a Wiredoor server using admin credentials and register this node.

This command allows you to connect to a Wiredoor instance and register the current node
via an interactive prompt. You'll be asked to provide:
  - Admin email and password
  - A name for the node (hostname by default)
  - Whether this node should act as a Gateway (able to expose other backends)
	- If your node is a gateway you'll need to define gateway network
  - Whether to route all traffic through the VPN (optional)

If a node is already configured locally, you will be prompted to overwrite it.

Example:
  wiredoor login --url https://my-wiredoor-server.local

Prompts will guide you through the registration and configuration process.`,
	Example: `  # Connect to a local Wiredoor instance
  wiredoor login --url https://192.168.50.134

  # Connect to a public Wiredoor server
  wiredoor login --url https://wiredoor.example.com`,
	Run: func(cmd *cobra.Command, args []string) {
		url, _ := cmd.Flags().GetString("url")

		if url == "" && !wiredoor.IsServerConfigSet() {
			fmt.Println("You must define Wiredoor server URL. Please use flag --url and try again.")
			return
		}

		if wiredoor.IsServerConfigSet() {
			doContinue := false

			survey.AskOne(&survey.Confirm{
				Message: "Another node is set, do you want to overwrite current config and set a new one?",
				Default: doContinue,
			}, &doContinue)

			if !doContinue {
				return
			}
		}

		var username, password, nodeName, subnet, iface string
		var isGateway, allowInternet bool

		hostname, _ := os.Hostname()
		defaultSubnet, _ := utils.DefaultSubnet()
		defaultInterface := utils.GetDefaultInterfaceName()

		survey.AskOne(&survey.Input{
			Message: "EMail:",
		}, &username, survey.WithValidator(survey.Required))

		survey.AskOne(&survey.Password{
			Message: "Password:",
		}, &password, survey.WithValidator(survey.Required))

		token, err := wiredoor.AdminLogin(url, username, password)

		if err != nil {
			printErrorAndExit(err, 1)
			return
		}

		survey.AskOne(&survey.Input{
			Message: "Node Name:",
			Default: hostname,
		}, &nodeName)

		survey.AskOne(&survey.Confirm{
			Message: "Is this node a Gateway?",
			Default: false,
		}, &isGateway)

		if isGateway {
			survey.AskOne(&survey.Input{
				Message: "Gateway Interface:",
				Default: defaultInterface,
			}, &iface, survey.WithValidator(survey.Required))
		}

		if isGateway {
			survey.AskOne(&survey.Input{
				Message: "Gateway CIDR Subnet:",
				Default: defaultSubnet,
			}, &subnet, survey.WithValidator(survey.Required))
		}

		survey.AskOne(&survey.Confirm{
			Message: "Send all internet traffic through the VPN?",
			Default: false,
		}, &allowInternet)

		networkConfig := wiredoor.GatewayNetwork{
			Subnet: subnet,
			Interface: iface,
		}

		node, err := wiredoor.ConfigureNode(url, token, wiredoor.NodeParams{
			Name:           nodeName,
			IsGateway:      isGateway,
			GatewayNetworks: []wiredoor.GatewayNetwork{networkConfig},
			AllowInternet:  allowInternet,
		})

		if err != nil {
			printErrorAndExit(err, 1)
			return
		}

		fmt.Printf("Node %s registered successfully!\n", node.Name)

		wiredoor.Connect(wiredoor.ConnectionConfig{})
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().String("url", "", "URL Domain or Server IP of Wiredoor instance")
}

func printErrorAndExit(err error, code int) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(code)
}
