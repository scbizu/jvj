package core

import "context"

type Router struct{}

func (r *Router) Route(_ context.Context, input string) (string, error) {
	return input, nil
}
