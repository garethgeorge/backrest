package types

import (
	"context"
	"reflect"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type mockTaskRunner struct {
	tasks.TaskRunner
}

func TestOsNotificationHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		hook    *v1.Hook
		vars    interface{}
		wantErr bool
	}{
		{
			name: "basic notification",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionOsNotification{
					ActionOsNotification: &v1.Hook_OsNotification{
						Template:      "test message",
						TitleTemplate: "test title",
						DeliveryMode:  0,
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
		{
			name: "with sound notification",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionOsNotification{
					ActionOsNotification: &v1.Hook_OsNotification{
						Template:      "test message",
						TitleTemplate: "test title",
						DeliveryMode:  2,
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
		{
			name: "with template variables",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionOsNotification{
					ActionOsNotification: &v1.Hook_OsNotification{
						Template:      "{{.test}} message",
						TitleTemplate: "{{.test}} title",
						DeliveryMode:  1,
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
		{
			name: "invalid template",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionOsNotification{
					ActionOsNotification: &v1.Hook_OsNotification{
						Template:      "{{.Value | printf \"%s\" | invalid_function}}",
						TitleTemplate: "test title",
						DeliveryMode:  0,
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: true,
		},
		{
			name: "invalid title template",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionOsNotification{
					ActionOsNotification: &v1.Hook_OsNotification{
						Template:      "test message",
						TitleTemplate: "{{.Value | printf \"%s\" | invalid_function}}",
						DeliveryMode:  0,
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: true,
		},
	}

	handler := &osNotificationHandler{}
	runner := &mockTaskRunner{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.Execute(context.Background(), tt.hook, tt.vars, runner, v1.Hook_CONDITION_SNAPSHOT_SUCCESS)
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOsNotificationHandler_Name(t *testing.T) {
	handler := &osNotificationHandler{}
	if got := handler.Name(); got != "osNotification" {
		t.Errorf("Name() = %v, want %v", got, "osNotification")
	}
}

func TestOsNotificationHandler_ActionType(t *testing.T) {
	handler := &osNotificationHandler{}
	want := reflect.TypeOf(&v1.Hook_ActionOsNotification{})
	if got := handler.ActionType(); got != want {
		t.Errorf("ActionType() = %v, want %v", got, want)
	}
}
