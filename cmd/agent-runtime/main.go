package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scbizu/jvj/internal/skills"
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
	if _, err := bootstrapBuiltinSkills(); err != nil {
		return err
	}
	_ = configPath
	return nil
}

func bootstrapBuiltinSkills() ([]skills.BuiltinSkillBundle, error) {
	return loadBuiltinSkills([]string{
		filepath.Join("skills", "builtins"),
		filepath.Join("..", "..", "skills", "builtins"),
	})
}

func loadBuiltinSkills(candidates []string) ([]skills.BuiltinSkillBundle, error) {
	for _, root := range candidates {
		if _, err := os.Stat(root); err == nil {
			bundles, err := skills.LoadBuiltinSkillBundles(root)
			if err != nil {
				return nil, err
			}
			if len(bundles) == 0 {
				return nil, errors.New("no built-in skill bundles found")
			}
			return bundles, nil
		}
	}
	return nil, os.ErrNotExist
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
