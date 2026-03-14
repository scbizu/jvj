package session

import "testing"

func TestManagerOpenInitializesCurrentSession(t *testing.T) {
	mgr := NewManager()

	current, err := mgr.Open("session-1")
	if err != nil {
		t.Fatalf("open current session: %v", err)
	}
	if current.Tape == nil {
		t.Fatal("expected current session tape")
	}
}

func TestManagerRejectsSecondActiveAttach(t *testing.T) {
	mgr := NewManager()

	if _, err := mgr.Open("session-1"); err != nil {
		t.Fatalf("open first session: %v", err)
	}
	if _, err := mgr.Open("session-2"); err == nil {
		t.Fatal("expected single-session rejection")
	}
}

func TestManagerCloseAllowsNextSession(t *testing.T) {
	mgr := NewManager()

	current, err := mgr.Open("session-1")
	if err != nil {
		t.Fatalf("open first session: %v", err)
	}

	if err := mgr.Close(current.ID); err != nil {
		t.Fatalf("close current session: %v", err)
	}

	next, err := mgr.Open("session-2")
	if err != nil {
		t.Fatalf("open next session: %v", err)
	}
	if next.ID != "session-2" {
		t.Fatalf("expected session-2, got %s", next.ID)
	}
}
