package core

import (
	"context"
	"strings"

	"github.com/scbizu/jvj/internal/tape"
	"github.com/scbizu/jvj/internal/tools"
)

type TapeWriter interface {
	Append(context.Context, string, tape.AppendInput) (*tape.Entry, error)
}

type AgentLoop struct {
	router *Router
	tape   TapeWriter
	exec   CommandExecutor
}

type CommandExecutor interface {
	Execute(context.Context, tools.CommandRequest) (*tools.ExecutionResult, error)
}

func NewAgentLoop(router *Router, tapeWriter TapeWriter, execs ...CommandExecutor) *AgentLoop {
	loop := &AgentLoop{router: router, tape: tapeWriter}
	if len(execs) > 0 {
		loop.exec = execs[0]
	}
	return loop
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
	if a.exec != nil && strings.HasPrefix(output, "cmd:") {
		script := strings.TrimSpace(strings.TrimPrefix(output, "cmd:"))
		result, err := a.exec.Execute(ctx, tools.CommandRequest{
			Goal: script,
			Plan: &tools.ExecutionPlan{
				Goal:  script,
				Steps: []tools.PlanStep{{Name: "run", Script: script}},
			},
		})
		if err != nil {
			return "", err
		}
		output = result.Stdout
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
