package session

import "sync"

type Manager struct {
	mu       sync.RWMutex
	sessions map[string]struct{}
}

func NewManager() *Manager {
	return &Manager{sessions: map[string]struct{}{}}
}
