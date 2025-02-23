package types

import (
	"context"
	"testing"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
	"github.com/gen2brain/beeep"
)

type mockTaskRunner struct {
	tasks.TaskRunner
}

func TestBeeepHandler_Execute(t *testing.T) {
	tests := []struct {
		name    string
		hook    *v1.Hook
		vars    interface{}
		wantErr bool
	}{
		{
			name: "basic notification",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionBeeep{
					ActionBeeep: &v1.Hook_Beeep{
						Template:      "test message",
						TitleTemplate: "test title",
						Frequency:     beeep.DefaultFreq,
						Duration:      int32(beeep.DefaultDuration),
						Icon:          "",
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
		{
			name: "custom frequency and duration",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionBeeep{
					ActionBeeep: &v1.Hook_Beeep{
						Template:      "test message",
						TitleTemplate: "test title",
						Frequency:     1000,
						Duration:      2000,
						Icon:          "assets/information.png",
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
		{
			name: "with template variables",
			hook: &v1.Hook{
				Action: &v1.Hook_ActionBeeep{
					ActionBeeep: &v1.Hook_Beeep{
						Template:      "{{.test}} message",
						TitleTemplate: "{{.test}} title",
					},
				},
			},
			vars:    map[string]interface{}{"test": "value"},
			wantErr: false,
		},
	}

	handler := &beeepHandler{}
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
