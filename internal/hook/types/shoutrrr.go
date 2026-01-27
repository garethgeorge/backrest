package types

import (
	"context"
	"fmt"
	"reflect"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/nicholas-fedor/shoutrrr"
	"go.uber.org/zap"
)

type shoutrrrHandler struct{}

func (shoutrrrHandler) Name() string {
	return "shoutrrr"
}

func (shoutrrrHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	payload, err := hookutil.RenderTemplateOrDefault(h.GetActionShoutrrr().GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	l := runner.Logger(ctx)

	l.Sugar().Infof("Sending shoutrrr message to %s", h.GetActionShoutrrr().GetShoutrrrUrl())
	l.Debug("Sending shoutrrr message", zap.String("payload", payload))

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
