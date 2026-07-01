package sqlitestore

import (
	"github.com/ncruces/go-sqlite3"
)

// init opens and closes an in-memory database in the background to force
// wazero's compile of the embedded SQLite WASM binary off the critical path
// of the first real sql.Open. The compile is guarded by a sync.Once inside
// go-sqlite3, so later opens just wait on the same Once if it isn't done yet.
func init() {
	go func() {
		if c, err := sqlite3.Open(":memory:"); err == nil {
			c.Close()
		}
	}()
}
