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

type telegramHandler struct{}

func (telegramHandler) Name() string {
	return "telegram"
}

func (telegramHandler) Execute(ctx context.Context, h *v1.Hook, vars interface{}, runner tasks.TaskRunner, event v1.Hook_Condition) error {
	t := h.GetActionTelegram()

	payload, err := hookutil.RenderTemplateOrDefault(t.GetTemplate(), hookutil.DefaultTemplate, vars)
	if err != nil {
		return fmt.Errorf("template rendering: %w", err)
	}

	l := runner.Logger(ctx)

	message := struct {
		ChatID    string `json:"chat_id"`
		Text      string `json:"text"`
		ParseMode string `json:"parse_mode"`
	}{
		ChatID:    t.GetChatId(),
		Text:      payload,
		ParseMode: "HTML",
	}

	l.Sugar().Infof("Sending telegram message to chat ID %s", t.GetChatId())
	l.Debug("Sending telegram message", zap.Any("message", message))

	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	postUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.GetBotToken())

	body, err := hookutil.PostRequest(postUrl, "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	l.Sugar().Debugf("Telegram response: %s", body)

	return nil
}

func (telegramHandler) ActionType() reflect.Type {
	return reflect.TypeOf(&v1.Hook_ActionTelegram{})
}

func init() {
	DefaultRegistry().RegisterHandler(&telegramHandler{})
}
