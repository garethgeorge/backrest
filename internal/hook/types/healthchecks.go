package types

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"path"
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
	baseURL := cmd.GetActionHealthchecks().GetWebhookUrl()
	u, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("parsing webhook URL: %w", err)
	}

	switch {
	case protoutil.IsStartCondition(event):
		u.Path = path.Join(u.Path, "start")
	case protoutil.IsErrorCondition(event):
		u.Path = path.Join(u.Path, "fail")
	case protoutil.IsLogCondition(event):
		u.Path = path.Join(u.Path, "log")
	}

	pingUrl := u.String()

	body, err := hookutil.PostRequest(pingUrl, "text/plain", bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("sending healthchecks message to %q: %w", pingUrl, err)
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
