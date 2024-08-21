package types

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"go.uber.org/zap"
)

type discordHandler struct{}

func (discordHandler) Name() string {
	return "discord"
}

func (discordHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	payload, err := hookutil.RenderTemplateOrDefault(h.GetActionDiscord().GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	l := runner.Logger(ctx)
	l.Sugar().Infof("Sending discord message to %s", h.GetActionDiscord().GetWebhookUrl())
	l.Debug("Sending discord message", zap.String("payload", payload))

	type Message struct {
		Content string `json:"content"`
	}

	request := Message{
		Content: payload, // leading newline looks better in discord.
	}

	requestBytes, _ := json.Marshal(request)
	body, err := hookutil.PostRequest(h.GetActionDiscord().GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	if err != nil {
		return fmt.Errorf("sending discord message to %q: %w", h.GetActionDiscord().GetWebhookUrl(), err)
	}
	zap.S().Debug("Discord response", zap.String("body", body))
	return nil
}

func (discordHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionDiscord{})
}

func init() {
	DefaultRegistry().RegisterHandler(&discordHandler{})
}
