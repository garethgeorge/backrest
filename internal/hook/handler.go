package hook

import (
	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/orchestrator/tasks"
)

type Handler interface {
	ShouldHandle(hook *v1.Hook) bool
	Execute(hook *v1.Hook, runner tasks.TaskRunner) error
}
