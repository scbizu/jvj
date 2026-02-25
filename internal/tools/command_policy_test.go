package tools

import "testing"

func TestCommandPolicy_RejectRawShellEval(t *testing.T) {
	p := NewCommandPolicy()
	err := p.Validate(CommandRequest{Raw: "rm -rf /"})
	if err == nil {
		t.Fatal("expected rejection for raw shell eval")
	}
}
