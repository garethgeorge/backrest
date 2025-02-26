package types

import (
	"context"
	"fmt"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/gen2brain/beeep"
	"reflect"
)

type osNotificationHandler struct{}

func (osNotificationHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	b := h.GetActionOsNotification()

	message, err := hookutil.RenderTemplateOrDefault(b.GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := hookutil.RenderTemplateOrDefault(b.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

	if b.GetDeliveryMode() != 0 {
		err = beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
		if err != nil {
			return fmt.Errorf("beeep: %w", err)
		}
	}

	err = beeep.Notify(title, message, "assets/information.png")
	if err != nil {
		return fmt.Errorf("beeep: %w", err)
	}

	return nil
}

func (osNotificationHandler) Name() string {
	return "osNotification"
}

func (osNotificationHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionOsNotification{})
}

func init() {
	DefaultRegistry().RegisterHandler(&osNotificationHandler{})
}
