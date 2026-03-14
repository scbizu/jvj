package skills

import (
	"os"
	"path/filepath"
)

type BuiltinSkillBundle struct {
	Name string
	Root string
}

func LoadBuiltinSkillBundles(root string) ([]BuiltinSkillBundle, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	bundles := make([]BuiltinSkillBundle, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillRoot := filepath.Join(root, entry.Name())
		if _, err := os.Stat(filepath.Join(skillRoot, "SKILL.md")); err != nil {
			continue
		}
		bundles = append(bundles, BuiltinSkillBundle{
			Name: entry.Name(),
			Root: skillRoot,
		})
	}
	return bundles, nil
}
