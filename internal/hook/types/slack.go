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

type slackHandler struct{}

func (slackHandler) Name() string {
	return "slack"
}

func (slackHandler) Execute(ctx context.Context, cmd *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {

	payload, err := hookutil.RenderTemplateOrDefault(cmd.GetActionSlack().GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	l := runner.Logger(ctx)

	var requestBytes []byte
	if json.Valid([]byte(payload)) {
		l.Sugar().Infof("Sending advanced slack message to %s", cmd.GetActionSlack().GetWebhookUrl())
		l.Sugar().Debugf("Sending advanced slack message: %s", payload)
		requestBytes = []byte(payload)
	} else {
		l.Sugar().Infof("Sending slack message to %s", cmd.GetActionSlack().GetWebhookUrl())
		l.Debug("Sending slack message", zap.String("payload", payload))
		request := struct {
			Text string `json:"text"`
		}{
			Text: "Backrest Notification\n" + payload,
		}
		requestBytes, _ = json.Marshal(request)
	}

	body, err := hookutil.PostRequest(cmd.GetActionSlack().GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	if err != nil {
		return fmt.Errorf("sending slack message to %q: %w", cmd.GetActionSlack().GetWebhookUrl(), err)
	}

	l.Debug("Slack response", zap.String("body", body))
	return nil
}

func (slackHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionSlack{})
}

func init() {
	DefaultRegistry().RegisterHandler(&slackHandler{})
}
