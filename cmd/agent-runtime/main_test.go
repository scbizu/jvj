package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestRunDoesNotRequireConfigFileToExist(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.toml")
	out := &bytes.Buffer{}

	if err := runWithIO([]string{missingPath}, "", strings.NewReader("hello\n"), out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "hello") {
		t.Fatalf("expected runtime output to include routed content, got %q", out.String())
	}
}

func TestRunTreatsExitAsRegularInput(t *testing.T) {
	configPath := writeTempConfig(t)
	out := &bytes.Buffer{}

	if err := runWithIO([]string{configPath}, "", strings.NewReader("exit\n"), out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.String(), "exit") {
		t.Fatalf("expected runtime output to include exit input, got %q", out.String())
	}
}

func writeTempConfig(t *testing.T) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "runtime.toml")
	if err := os.WriteFile(configPath, []byte("[server]\nport = 0\n"), 0o600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return configPath
}
