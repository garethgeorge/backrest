package hook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func (h *Hook) doSlack(cmd *v1.Hook_ActionSlack, vars HookVars, output io.Writer) error {
	payload, err := h.renderTemplateOrDefault(cmd.ActionSlack.GetTemplate(), defaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	type Message struct {
		Text string `json:"text"`
	}

	request := Message{
		Text: "Backrest Notification\n" + payload, // leading newline looks better in discord.
	}

	requestBytes, _ := json.Marshal(request)

	fmt.Fprintf(output, "Sending Slack message to %s\n---- payload ----\n", cmd.ActionSlack.GetWebhookUrl())
	output.Write(requestBytes)

	_, err = post(cmd.ActionSlack.GetWebhookUrl(), "application/json", bytes.NewReader(requestBytes))
	return err
}
