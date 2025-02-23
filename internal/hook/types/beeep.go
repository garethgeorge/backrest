package types

import (
	"context"
	"fmt"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/gen2brain/beeep"
	"log"
	"reflect"
)

type beeepHandler struct{}

func (beeepHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	b := h.GetActionBeeep()

	message, err := hookutil.RenderTemplateOrDefault(b.GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := hookutil.RenderTemplateOrDefault(b.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

	frequency := b.GetFrequency()
	if frequency == 0 {
		frequency = beeep.DefaultFreq
	}

	duration := b.GetDuration()
	if duration == 0 {
		duration = int32(beeep.DefaultDuration)
	}

	icon := b.GetIcon()
	if icon == "" {
		icon = "assets/information.png"
	}

	log.Printf("Sending beeep notification icon %s", icon)
	err = beeep.Alert(title, message, icon)
	if err != nil {
		return fmt.Errorf("beeep: %w", err)
	}

	err = beeep.Beep(b.GetFrequency(), int(b.GetDuration()))
	if err != nil {
		return fmt.Errorf("beeep: %w", err)
	}
	return nil
}

func (beeepHandler) Name() string {
	return "beeep"
}

func (beeepHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionBeeep{})
}

func init() {
	DefaultRegistry().RegisterHandler(&beeepHandler{})
}
