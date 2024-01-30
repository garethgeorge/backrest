package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func (h *Hook) doDiscord(cmd *v1.Hook_ActionDiscord, vars HookVars, output io.Writer) error {
	payload, err := h.renderTemplateOrDefault(cmd.ActionDiscord.GetTemplate(), defaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	type Message struct {
		Content string `json:"content"`
	}

	request := Message{
		Content: "Backrest Notification\n" + payload, // leading newline looks better in discord.
	}

	requestBytes, _ := json.Marshal(request)

	_, err = post(cmd.ActionDiscord.GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	return err
}
