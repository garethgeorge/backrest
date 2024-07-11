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
)

type slackHandler struct{}

func (slackHandler) Name() string {
	return "slack"
}

func (slackHandler) Execute(ctx context.Context, cmd *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	payload, err := hookutil.RenderTemplateOrDefault(cmd.GetActionSlack().GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	writer := runner.RawLogWriter(ctx)
	fmt.Fprintf(writer, "Sending slack message to %s\n", cmd.GetActionSlack().GetWebhookUrl())
	fmt.Fprintf(writer, "---- payload ----\n%s\n", payload)

	type Message struct {
		Text string `json:"text"`
	}

	request := Message{
		Text: "Backrest Notification\n" + payload, // leading newline looks better in discord.
	}

	requestBytes, _ := json.Marshal(request)

	_, err = hookutil.PostRequest(cmd.GetActionSlack().GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	return err
}

func (slackHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionSlack{})
}

func init() {
	DefaultRegistry().RegisterHandler(&slackHandler{})
}
