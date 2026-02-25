package tools

import "errors"

type CommandRequest struct {
	Raw  string
	Argv []string
}

type CommandPolicy struct{}

func NewCommandPolicy() *CommandPolicy {
	return &CommandPolicy{}
}

func (p *CommandPolicy) Validate(req CommandRequest) error {
	if req.Raw != "" {
		return errors.New("raw shell eval is forbidden")
	}
	if len(req.Argv) == 0 {
		return errors.New("argv is required")
	}
	return nil
}
