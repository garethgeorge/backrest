package types

import (
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
)

func TestTelegramHandlerRegistration(t *testing.T) {
	// Test that the telegram handler is registered
	hook := &v1.Hook{
		Action: &v1.Hook_ActionTelegram{
			ActionTelegram: &v1.Hook_Telegram{
				BotToken: "test-token",
				ChatId:   "12345",
				Template: "test template",
			},
		},
	}

	handler, err := DefaultRegistry().GetHandler(hook)
	if err != nil {
		t.Fatalf("Failed to get telegram handler: %v", err)
	}

	if handler.Name() != "telegram" {
		t.Errorf("Expected handler name to be 'telegram', got '%s'", handler.Name())
	}
}
