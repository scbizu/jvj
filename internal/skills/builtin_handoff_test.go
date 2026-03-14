package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuiltinHandoffSkillBundleHasSkillMarkdown(t *testing.T) {
	if _, err := os.Stat(filepath.Join("..", "..", "skills", "builtins", "handoff", "SKILL.md")); err != nil {
		t.Fatalf("expected built-in handoff skill bundle: %v", err)
	}
}

func TestBuiltinHandoffSkillBundleUsesValidSkillName(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "skills", "builtins", "handoff", "SKILL.md"))
	if err != nil {
		t.Fatalf("read SKILL.md: %v", err)
	}

	if !strings.Contains(string(content), "name: handoff") {
		t.Fatal("expected handoff skill frontmatter")
	}
}
