package hook

import (
	"fmt"
	"io"

	"github.com/containrrr/shoutrrr"
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func (h *Hook) doShoutrrr(cmd *v1.Hook_ActionShoutrrr, vars HookVars, output io.Writer) error {
	payload, err := h.renderTemplateOrDefault(cmd.ActionShoutrrr.GetTemplate(), defaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	fmt.Fprintf(output, "Sending notification to %s\nContents:\n", cmd.ActionShoutrrr.GetShoutrrrUrl())
	output.Write([]byte(payload))

	if err := shoutrrr.Send(cmd.ActionShoutrrr.GetShoutrrrUrl(), payload); err != nil {
		return fmt.Errorf("send notification to %q: %w", cmd.ActionShoutrrr.GetShoutrrrUrl(), err)
	}
	return nil
}
