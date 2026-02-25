package core

import "context"

type AgentLoop struct {
	router *Router
}

func NewAgentLoop(router *Router) *AgentLoop {
	return &AgentLoop{router: router}
}

func (a *AgentLoop) Run(ctx context.Context, input string) (string, error) {
	return a.router.Route(ctx, input)
}
