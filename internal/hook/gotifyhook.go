package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func (h *Hook) doGotify(cmd *v1.Hook_ActionGotify, vars HookVars, output io.Writer) error {
	payload, err := h.renderTemplateOrDefault(cmd.ActionGotify.GetTemplate(), defaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	title, err := h.renderTemplateOrDefault(cmd.ActionGotify.GetTitleTemplate(), "Backrest Event", vars)
	if err != nil {
		return fmt.Errorf("title template rendering: %w", err)
	}

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

	baseUrl := strings.Trim(cmd.ActionGotify.GetBaseUrl(), "/")

	postUrl := fmt.Sprintf(
		"%s/message?token=%s",
		baseUrl,
		url.QueryEscape(cmd.ActionGotify.GetToken()))

	fmt.Fprintf(output, "Sending gotify message to %s\n", postUrl)
	fmt.Fprintf(output, "---- payload ----\n")
	output.Write(b)

	body, err := post(postUrl, "application/json", bytes.NewReader(b))

	if err != nil {
		return fmt.Errorf("send gotify message: %w", err)
	}

	if body != "" {
		output.Write([]byte(body))
	}

	return nil
}
