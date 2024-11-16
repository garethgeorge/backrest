package sqlitestore

import (
	"fmt"

	"go.uber.org/zap"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func applySqliteMigrations(db *sqlite.Conn, migrations []string) error {
	var version int
	if err := sqlitex.ExecuteTransient(db, "PRAGMA user_version", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			version = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("getting database schema version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		zap.L().Info("Applying sqlite schema migration ", zap.Int("version", i+1))
		// Add a pragma statement to set the user_version to the migration number,
		// the whole script including the pragma statement is executed within a single
		// transaction to ensure that the migration is atomic.
		scriptWithPragma := fmt.Sprintf("PRAGMA user_version = %d;\n%s", i+1, migrations[i])

		if err := sqlitex.ExecuteScript(db, scriptWithPragma, &sqlitex.ExecOptions{}); err != nil {
			return fmt.Errorf("applying migration %d: %w", i, err)
		}
	}

	return nil
}
