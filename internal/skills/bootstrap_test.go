package skills

import (
	"path/filepath"
	"testing"
)

func TestLoadBuiltinSkillBundlesFindsHandoff(t *testing.T) {
	bundles, err := LoadBuiltinSkillBundles(filepath.Join("..", "..", "skills", "builtins"))
	if err != nil {
		t.Fatalf("load built-in bundles: %v", err)
	}

	if len(bundles) == 0 || bundles[0].Name == "" {
		t.Fatal("expected at least one built-in skill bundle")
	}
}

func TestLoadBuiltinSkillBundlesSkipsMissingSkillMarkdown(t *testing.T) {
	bundles, err := LoadBuiltinSkillBundles("testdata/missing-skill-md")
	if err != nil {
		t.Fatalf("load built-in bundles: %v", err)
	}
	if len(bundles) != 0 {
		t.Fatalf("expected no valid bundles, got %+v", bundles)
	}
}

func TestLoadBuiltinSkillBundlesSkipsNonSkillDirectories(t *testing.T) {
	bundles, err := LoadBuiltinSkillBundles("testdata/mixed-root")
	if err != nil {
		t.Fatalf("load built-in bundles: %v", err)
	}

	if len(bundles) != 1 || bundles[0].Name != "handoff" {
		t.Fatalf("expected only handoff bundle, got %+v", bundles)
	}
}
