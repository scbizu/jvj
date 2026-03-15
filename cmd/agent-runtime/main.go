package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/scbizu/jvj/internal/core"
	"github.com/scbizu/jvj/internal/session"
	"github.com/scbizu/jvj/internal/skills"
	"github.com/scbizu/jvj/internal/tape"
	"github.com/scbizu/jvj/internal/tools"
	"github.com/spf13/cobra"
)

var version = "dev"

type runtimeSessionManager interface {
	Open(string) (*session.Session, error)
	Close(string) error
}

type runtimeLoop interface {
	Run(context.Context, string, string) (string, error)
}

type runtimeDeps struct {
	newSessionManager func() runtimeSessionManager
	newLoop           func() runtimeLoop
}

func run(args []string, configPath string) error {
	return runWithIO(args, configPath, os.Stdin, os.Stdout)
}

func runWithIO(args []string, configPath string, in io.Reader, out io.Writer) error {
	return runWithDeps(args, configPath, in, out, runtimeDeps{
		newSessionManager: func() runtimeSessionManager {
			return session.NewManager()
		},
		newLoop: func() runtimeLoop {
			return core.NewAgentLoop(
				&core.Router{},
				tape.NewService(tape.NewInMemoryStore()),
				tools.NewRegistry(),
			)
		},
	})
}

func runWithDeps(args []string, configPath string, in io.Reader, out io.Writer, deps runtimeDeps) (err error) {
	if configPath == "" && len(args) > 0 {
		configPath = args[0]
	}
	if configPath == "" {
		return errors.New("config path is required")
	}
	if _, err := os.Stat(configPath); err != nil {
		return err
	}
	if _, err := bootstrapBuiltinSkills(); err != nil {
		return err
	}

	sessionManager := deps.newSessionManager()
	activeSession, err := sessionManager.Open("interactive")
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sessionManager.Close(activeSession.ID); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	loop := deps.newLoop()

	scanner := bufio.NewScanner(in)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == "exit" {
			break
		}

		output, err := loop.Run(ctx, activeSession.ID, line)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, output); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

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
			return runWithIO(args, configPath, cmd.InOrStdin(), cmd.OutOrStdout())
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
