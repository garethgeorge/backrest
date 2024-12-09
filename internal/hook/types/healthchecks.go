package types

import (
    "bytes"
    "context"
    "fmt"
    "reflect"

    v1 "github.com/garethgeorge/backrest/gen/go/v1"
    "github.com/garethgeorge/backrest/internal/hook/hookutil"
    "github.com/garethgeorge/backrest/internal/orchestrator/tasks"
    "github.com/garethgeorge/backrest/internal/protoutil"
    "go.uber.org/zap"
)

type healthchecksHandler struct{}

func (healthchecksHandler) Name() string {
    return "healthchecks"
}

func (healthchecksHandler) Execute(ctx context.Context, cmd *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
    payload, err := hookutil.RenderTemplateOrDefault(cmd.GetActionHealthchecks().GetTemplate(), hookutil.DefaultTemplate, vars)
    if err != nil {
        return fmt.Errorf("template rendering: %w", err)
    }

    l := runner.Logger(ctx)
    l.Sugar().Infof("Sending healthchecks message to %s", cmd.GetActionHealthchecks().GetWebhookUrl())
    l.Debug("Sending healthchecks message", zap.String("payload", payload))

    PingUrl := cmd.GetActionHealthchecks().GetWebhookUrl()

	// Send a "start" signal to healthchecks.io when the hook is starting.
    if protoutil.IsStartCondition(event) {
        PingUrl += "/start"
    }

	// Send a "fail" signal to healthchecks.io when the hook is failing.
    if protoutil.IsErrorCondition(event) {
        PingUrl += "/fail"
    }

	// Send a "log" signal to healthchecks.io when the hook is ending.
	if protoutil.IsLogCondition(event) {
        PingUrl += "/log"
    }

    body, err := hookutil.PostRequest(PingUrl, "text/plain", bytes.NewBufferString(payload))
    if err != nil {
        return fmt.Errorf("sending healthchecks message to %q: %w", PingUrl, err)
    }

    l.Debug("Healthchecks response", zap.String("body", body))
    return nil
}

func (healthchecksHandler) ActionType() reflect.Type {
    return reflect.TypeOf(&v1.Hook_ActionHealthchecks{})
}

func init() {
    DefaultRegistry().RegisterHandler(&healthchecksHandler{})
}