package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func run(args []string, configPath string) error {
	if configPath == "" && len(args) > 0 {
		configPath = args[0]
	}
	if configPath == "" {
		return errors.New("config path is required")
	}
	_ = configPath
	return nil
}

func newRunCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:          "run [config-path]",
		Args:         cobra.MaximumNArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, configPath)
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use: "agent-runtime",
	}
	root.AddCommand(newRunCmd())
	root.AddCommand(newVersionCmd())
	return root
}

func Execute() error {
	return newRootCmd().Execute()
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
