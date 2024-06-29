package hook

import (
	"context"
	"errors"
	"reflect"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

var ErrHandlerNotFound = errors.New("handler not found")

// defaultRegistry is the default handler registry.
var defaultRegistry = &HandlerRegistry{
	actionHandlers: make(map[reflect.Type]Handler),
}

func DefaultRegistry() *HandlerRegistry {
	return defaultRegistry
}

type HandlerRegistry struct {
	actionHandlers map[reflect.Type]Handler
}

// RegisterHandler registers a handler with the default registry.
func (r *HandlerRegistry) RegisterHandler(handler Handler) {
	r.actionHandlers[handler.ActionType()] = handler
}

func (r *HandlerRegistry) GetHandler(hook *v1.Hook) (Handler, error) {
	handler, ok := r.actionHandlers[reflect.TypeOf(hook.Action)]
	if !ok {
		return nil, ErrHandlerNotFound
	}
	return handler, nil
}

type Handler interface {
	Execute(ctx context.Context, hook *v1.Hook, vars interface{}, runner tasks.TaskRunner) error
	ActionType() reflect.Type
}
