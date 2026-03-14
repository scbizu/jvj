package main

import (
	"bytes"
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
	cmd := newRunCmd()
	cmd.SetArgs([]string{"--config", "config.toml"})
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
