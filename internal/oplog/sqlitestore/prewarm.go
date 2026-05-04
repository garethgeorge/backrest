package sqlitestore

import (
	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"
)

// init kicks off wazero's compilation of the embedded SQLite WASM binary in
// the background. The compile is guarded by a sync.Once inside go-sqlite3, so
// any later sql.Open call simply waits on the same Once if it isn't done yet.
// The goal is to move this multi-second cost off the critical path of the
// first sqlite open, which otherwise inflates startup latency and burns into
// per-test deadlines (notably under -race).
func init() {
	go sqlite3.Initialize()
}
