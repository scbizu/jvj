package core

import (
	"context"

	"github.com/scbizu/jvj/internal/tape"
)

type TapeWriter interface {
	Append(context.Context, string, tape.AppendInput) (*tape.Entry, error)
}

type AgentLoop struct {
	router *Router
	tape   TapeWriter
}

func NewAgentLoop(router *Router, tapeWriter TapeWriter) *AgentLoop {
	return &AgentLoop{router: router, tape: tapeWriter}
}

func (a *AgentLoop) Run(ctx context.Context, sessionID, input string) (string, error) {
	if _, err := a.tape.Append(ctx, sessionID, tape.AppendInput{
		Kind:    tape.EntryUser,
		Content: input,
		Actor:   "user",
	}); err != nil {
		return "", err
	}

	output, err := a.router.Route(ctx, input)
	if err != nil {
		return "", err
	}
	if _, err := a.tape.Append(ctx, sessionID, tape.AppendInput{
		Kind:    tape.EntryAssistant,
		Content: output,
		Actor:   "agent",
	}); err != nil {
		return "", err
	}
	return output, nil
}
