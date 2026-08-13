package types

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"go.uber.org/zap"
)

const pushoverAPIURL = "https://api.pushover.net/1/messages.json"

type pushoverHandler struct{}

func (pushoverHandler) Name() string {
	return "pushover"
}

func (pushoverHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	p := h.GetActionPushover()

	payload, err := hookutil.RenderTemplateOrDefault(p.GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := hookutil.RenderTemplateOrDefault(p.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

	l := runner.Logger(ctx)
	l.Sugar().Infof("Sending Pushover notification for user key %s", p.GetUserKey())
	l.Debug("Sending Pushover notification", zap.String("title", title))

	form := url.Values{}
	form.Set("token", p.GetToken())
	form.Set("user", p.GetUserKey())
	form.Set("message", payload)
	form.Set("title", title)
	form.Set("priority", strconv.Itoa(int(p.GetPriority())))

	resp, err := http.PostForm(pushoverAPIURL, form)
	if err != nil {
		return fmt.Errorf("sending Pushover notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Pushover API returned unexpected status %d: %s", resp.StatusCode, resp.Status)
	}

	l.Sugar().Debugf("Pushover notification sent successfully (status %d)", resp.StatusCode)
	return nil
}

func (pushoverHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionPushover{})
}

func init() {
	DefaultRegistry().RegisterHandler(&pushoverHandler{})
}
