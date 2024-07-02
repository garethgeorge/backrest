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
	"github.com/garethgeorge/backrest/internal/hook"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type gotifyHandler struct{}

func (gotifyHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner) error {
	g := h.GetActionGotify()

	payload, err := hook.RenderTemplateOrDefault(g.GetTemplate(), hook.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := hook.RenderTemplateOrDefault(g.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

	output := runner.RawLogWriter(ctx)

	message := struct {
		Message  string `json:"message"`
		Title    string `json:"title"`
		Priority int    `json:"priority"`
	}{
		Title:    title,
		Priority: 5,
		Message:  payload,
	}

	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	baseUrl := strings.Trim(g.GetBaseUrl(), "/")

	postUrl := fmt.Sprintf(
		"%s/message?token=%s",
		baseUrl,
		url.QueryEscape(g.GetToken()))

	fmt.Fprintf(output, "Sending gotify message to %s\n", postUrl)
	fmt.Fprintf(output, "---- payload ----\n")
	output.Write(b)

	body, err := hook.PostRequest(postUrl, "application/json", bytes.NewReader(b))

	if err != nil {
		return fmt.Errorf("send gotify message: %w", err)
	}

	if body != "" {
		output.Write([]byte(body))
	}

	return nil
}

func (gotifyHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionGotify{})
}
