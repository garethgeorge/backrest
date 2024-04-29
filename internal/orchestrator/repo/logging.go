package repo

import (
	"context"

	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/pkg/restic"
)

// pipeResticLogsToWriter sets the restic logger to write to the provided writer.
// returns a new context with the logger set and a function to flush the logs.
func forwardResticLogs(ctx context.Context) (context.Context, func()) {
	capture := ioutil.NewOutputCapturer(8 * 1024)
	return restic.ContextWithLogger(ctx, capture), func() {
		writer := logging.WriterFromContext(ctx)
		if writer == nil {
			return
		}
		_, _ = writer.Write(capture.Bytes())
	}
}
