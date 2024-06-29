package types

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type discordHandler struct{}

func (discordHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	payload, err := hook.RenderTemplateOrDefault(h.GetActionDiscord().GetTemplate(), hook.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	writer := runner.RawLogWriter(ctx)
	fmt.Fprintf(writer, "Sending discord message to %s\n", h.GetActionDiscord().GetWebhookUrl())
	fmt.Fprintf(writer, "---- payload ----\n%s\n", payload)

	type Message struct {
		Content string `json:"content"`
	}

	request := Message{
		Content: payload, // leading newline looks better in discord.
	}

	requestBytes, _ := json.Marshal(request)
	_, err = hook.PostRequest(h.GetActionDiscord().GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	return err
}

func (discordHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionDiscord{})
}

func init() {
	hook.DefaultRegistry().RegisterHandler(&discordHandler{})
}
