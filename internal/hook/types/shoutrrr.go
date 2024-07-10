package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/containrrr/shoutrrr"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type shoutrrrHandler struct{}

func (shoutrrrHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	payload, err := hookutil.RenderTemplateOrDefault(h.GetActionShoutrrr().GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	writer := runner.RawLogWriter(ctx)
	fmt.Fprintf(writer, "Sending shoutrrr message to %s\n", h.GetActionShoutrrr().GetShoutrrrUrl())
	fmt.Fprintf(writer, "---- payload ----\n%s\n", payload)

	if err := shoutrrr.Send(h.GetActionShoutrrr().GetShoutrrrUrl(), payload); err != nil {
		return fmt.Errorf("sending shoutrrr message to %q: %w", h.GetActionShoutrrr().GetShoutrrrUrl(), err)
	}

	return nil
}

func (shoutrrrHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionShoutrrr{})
}

func init() {
	DefaultRegistry().RegisterHandler(&shoutrrrHandler{})
}
