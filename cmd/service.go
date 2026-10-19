package cmd

import "github.com/spf13/cobra"

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the local Wiredoor system service",
	Args:  cobra.NoArgs,
}

func newServiceAction(name, description string, action func() error) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: description,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := action(); err != nil {
				return err
			}
			cmd.Printf("Service %s completed.\n", name)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(serviceCmd)
}
