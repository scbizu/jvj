package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scbizu/jvj/internal/session"
)

func TestRunCmdConfigPathRequired(t *testing.T) {
	cmd := newRunCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when config path is missing")
	}
}

func TestRunCmdConfigFlagWorks(t *testing.T) {
	configPath := writeTempConfig(t)
	cmd := newRunCmd()
	cmd.SetArgs([]string{"--config", configPath})
	cmd.SetIn(strings.NewReader(""))
	cmd.SetOut(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCmdPrintsPlaceholder(t *testing.T) {
	cmd := newVersionCmd()
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "dev") {
		t.Fatalf("expected placeholder version, got: %q", buf.String())
	}
}

func TestRuntimeInitializationPreloadsBuiltinSkills(t *testing.T) {
	bundles, err := bootstrapBuiltinSkills()
	if err != nil {
		t.Fatalf("load bundles: %v", err)
	}

	if len(bundles) == 0 {
		t.Fatal("expected built-in skills to preload during init")
	}
}

func TestLoadBuiltinSkillsRejectsEmptyBuiltinRoots(t *testing.T) {
	if _, err := loadBuiltinSkills([]string{t.TempDir()}); err == nil {
		t.Fatal("expected empty builtin roots to fail")
	}
}

func TestRunProcessesInteractiveInput(t *testing.T) {
	configPath := writeTempConfig(t)
	in := strings.NewReader("hello\nexit\n")
	out := &bytes.Buffer{}

	if err := runWithIO([]string{configPath}, "", in, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected runtime output to include routed content, got %q", out.String())
	}
}

func TestRunRejectsMissingConfigFile(t *testing.T) {
	err := runWithIO([]string{"missing.toml"}, "", strings.NewReader(""), &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected missing config file to fail")
	}
}

func TestRunStopsOnExitCommand(t *testing.T) {
	configPath := writeTempConfig(t)
	in := strings.NewReader("exit\nignored\n")
	out := &bytes.Buffer{}

	if err := runWithIO([]string{configPath}, "", in, out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Len() != 0 {
		t.Fatalf("expected exit to stop without emitting turn output, got %q", out.String())
	}
}

func TestRunReturnsSessionCloseError(t *testing.T) {
	configPath := writeTempConfig(t)
	closeErr := errors.New("close failed")

	err := runWithDeps([]string{configPath}, "", strings.NewReader("hello\n"), &bytes.Buffer{}, runtimeDeps{
		newSessionManager: func() runtimeSessionManager {
			return &stubSessionManager{
				session:  &session.Session{ID: "runtime", Attached: true},
				closeErr: closeErr,
			}
		},
		newLoop: func() runtimeLoop {
			return stubRuntimeLoop{output: "hello"}
		},
	})
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
}

func TestRunCancelsContextOnExit(t *testing.T) {
	configPath := writeTempConfig(t)
	loop := &capturingRuntimeLoop{output: "hello"}

	if err := runWithDeps([]string{configPath}, "", strings.NewReader("hello\n"), &bytes.Buffer{}, runtimeDeps{
		newSessionManager: func() runtimeSessionManager {
			return &stubSessionManager{
				session: &session.Session{ID: "runtime", Attached: true},
			}
		},
		newLoop: func() runtimeLoop {
			return loop
		},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if loop.ctx == nil {
		t.Fatal("expected loop to receive a context")
	}
	select {
	case <-loop.ctx.Done():
	default:
		t.Fatal("expected context to be canceled when run exits")
	}
}

type stubSessionManager struct {
	session  *session.Session
	openErr  error
	closeErr error
}

func (s *stubSessionManager) Open(id string) (*session.Session, error) {
	if s.openErr != nil {
		return nil, s.openErr
	}
	if s.session != nil {
		return s.session, nil
	}
	return &session.Session{ID: id, Attached: true}, nil
}

func (s *stubSessionManager) Close(string) error {
	return s.closeErr
}

type stubRuntimeLoop struct {
	output string
	err    error
}

func (s stubRuntimeLoop) Run(context.Context, string, string) (string, error) {
	return s.output, s.err
}

type capturingRuntimeLoop struct {
	ctx    context.Context
	output string
	err    error
}

func (c *capturingRuntimeLoop) Run(ctx context.Context, _ string, _ string) (string, error) {
	c.ctx = ctx
	return c.output, c.err
}

func writeTempConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "runtime.toml")
	if err := os.WriteFile(configPath, []byte("[server]\nport = 0\n"), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return configPath
}
