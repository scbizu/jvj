package tools

import (
	"strings"
	"testing"
)

func TestScriptBuilderBuildsOneShotScriptWithSafetyHeader(t *testing.T) {
	builder := NewScriptBuilder("/tmp")

	artifact, err := builder.Build(ExecutionPlan{
		Goal:  "run tests",
		Steps: []PlanStep{{Name: "test", Script: "go test ./..."}},
	})
	if err != nil {
		t.Fatalf("build script: %v", err)
	}

	if !strings.Contains(artifact.Content, "set -euo pipefail") {
		t.Fatal("expected safety header in script")
	}
}

func TestScriptBuilderIncludesPlanStepsInOrder(t *testing.T) {
	builder := NewScriptBuilder("/tmp")

	artifact, err := builder.Build(ExecutionPlan{
		Goal: "fix",
		Steps: []PlanStep{
			{Name: "check", Script: "pwd"},
			{Name: "apply", Script: "go test ./..."},
		},
	})
	if err != nil {
		t.Fatalf("build script: %v", err)
	}

	if strings.Index(artifact.Content, "pwd") > strings.Index(artifact.Content, "go test ./...") {
		t.Fatal("expected script steps to preserve plan order")
	}
}
