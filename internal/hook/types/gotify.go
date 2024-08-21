package types

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/hook/hookutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"go.uber.org/zap"
)

type gotifyHandler struct{}

func (gotifyHandler) Name() string {
	return "gotify"
}

func (gotifyHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	g := h.GetActionGotify()

	payload, err := hookutil.RenderTemplateOrDefault(g.GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := hookutil.RenderTemplateOrDefault(g.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

	l := runner.Logger(ctx)

	message := struct {
		Message  string `json:"message"`
		Title    string `json:"title"`
		Priority int    `json:"priority"`
	}{
		Title:    title,
		Priority: 5,
		Message:  payload,
	}

	l.Sugar().Infof("Sending gotify message to %s", g.GetBaseUrl())
	l.Debug("Sending gotify message", zap.Any("message", message))

	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	baseUrl := strings.Trim(g.GetBaseUrl(), "/")

	postUrl := fmt.Sprintf(
		"%s/message?token=%s",
		baseUrl,
		url.QueryEscape(g.GetToken()))

	body, err := hookutil.PostRequest(postUrl, "application/json", bytes.NewReader(b))

	if err != nil {
		return fmt.Errorf("send gotify message: %w", err)
	}

	l.Sugar().Debugf("Gotify response: %s", body)

	return nil
}

func (gotifyHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionGotify{})
}

func init() {
	DefaultRegistry().RegisterHandler(&gotifyHandler{})
}
