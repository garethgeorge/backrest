package repo

import (
	"context"
	"fmt"

	"github.com/garethgeorge/backrest/internal/ioutil"
	"github.com/garethgeorge/backrest/internal/orchestrator/logging"
	"github.com/garethgeorge/backrest/pkg/restic"
)

// pipeResticLogsToWriter sets the restic logger to write to the provided writer.
// returns a new context with the logger set and a function to flush the logs.
func forwardResticLogs(ctx context.Context) (context.Context, func()) {
	writer := logging.WriterFromContext(ctx)
	if writer == nil {
		return ctx, func() {}
	}
	limitWriter := &ioutil.LimitWriter{W: writer, N: 64 * 1024}
	prefixWriter := &ioutil.LinePrefixer{W: limitWriter, Prefix: []byte("[restic] ")}
	return restic.ContextWithLogger(ctx, prefixWriter), func() {
		if limitWriter.D > 0 {
			fmt.Fprintf(prefixWriter, "... Output truncated, %d bytes dropped\n", limitWriter.D)
		}
		prefixWriter.Close()
	}
}
