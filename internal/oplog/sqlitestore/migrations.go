package sqlitestore

import (
	"fmt"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const sqlSchemaVersion = 4

var sqlSchema = fmt.Sprintf(`
PRAGMA user_version = %d;

CREATE TABLE IF NOT EXISTS system_info (version INTEGER NOT NULL);
INSERT INTO system_info (version)
SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM system_info);

CREATE TABLE operations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ogid INTEGER NOT NULL,
	original_id INTEGER NOT NULL,
	original_flow_id INTEGER NOT NULL,
	modno INTEGER NOT NULL,
	flow_id INTEGER NOT NULL,
	start_time_ms INTEGER NOT NULL,
	status INTEGER NOT NULL,
	snapshot_id STRING NOT NULL,
	operation BLOB NOT NULL,
	FOREIGN KEY (ogid) REFERENCES operation_groups (ogid)
);
CREATE INDEX operation_ogid ON operations (ogid);
CREATE INDEX operation_snapshot_id ON operations (snapshot_id);
CREATE INDEX operation_flow_id ON operations (flow_id);
CREATE INDEX operation_start_time_ms ON operations (start_time_ms);
CREATE INDEX operation_original_id ON operations (ogid, original_id);
CREATE INDEX operation_original_flow_id ON operations (ogid, original_flow_id);

CREATE TABLE operation_groups (
	ogid INTEGER PRIMARY KEY AUTOINCREMENT,
	instance_id STRING NOT NULL,
	repo_guid STRING NOT NULL,
	repo_id STRING NOT NULL,
	plan_id STRING NOT NULL
);
CREATE INDEX group_repo_instance ON operation_groups (repo_id, instance_id);
CREATE INDEX group_repo_guid ON operation_groups (repo_guid);
CREATE INDEX group_instance ON operation_groups (instance_id);
`, sqlSchemaVersion)

func applySqliteMigrations(store *SqliteStore, conn *sqlite.Conn, dbPath string) error {
	var version int
	if err := sqlitex.ExecuteTransient(conn, "PRAGMA user_version", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			version = stmt.ColumnInt(0)
			return nil
		},
	}); err != nil {
		return fmt.Errorf("getting database schema version: %w", err)
	}

	if version == sqlSchemaVersion {
		return nil
	}

	zap.S().Infof("applying oplog sqlite schema migration from storage schema %d to %d", version, sqlSchemaVersion)

	if err := withSqliteTransaction(conn, func() error {
		// Check if operations table exists and rename it
		var hasOperationsTable bool
		if err := sqlitex.ExecuteTransient(conn, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='operations'", &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				hasOperationsTable = stmt.ColumnInt(0) > 0
				return nil
			},
		}); err != nil {
			return fmt.Errorf("checking for operations table: %w", err)
		}

		if hasOperationsTable {
			zap.S().Info("renaming existing operations table to operations_old as a backup")
			if err := sqlitex.ExecuteTransient(conn, "ALTER TABLE operations RENAME TO operations_old", nil); err != nil {
				return fmt.Errorf("renaming operations table: %w", err)
			}

			// drop all tables that aren't operations_old
			drop_tables := []string{
				"operation_groups",
				"operations",
			}
			for _, table := range drop_tables {
				if err := sqlitex.ExecuteTransient(conn, fmt.Sprintf("DROP TABLE IF EXISTS %s", table), nil); err != nil {
					return fmt.Errorf("dropping table %s: %w", table, err)
				}
			}

			// drop all indexes
			indexes := []string{}
			if err := sqlitex.ExecuteTransient(conn, "SELECT name FROM sqlite_master WHERE type='index'", &sqlitex.ExecOptions{
				ResultFunc: func(stmt *sqlite.Stmt) error {
					indexes = append(indexes, stmt.ColumnText(0))
					return nil
				},
			}); err != nil {
				return fmt.Errorf("dropping indexes: %w", err)
			}
			for _, index := range indexes {
				if err := sqlitex.ExecuteTransient(conn, fmt.Sprintf("DROP INDEX IF EXISTS %s", index), nil); err != nil {
					return fmt.Errorf("dropping index %s: %w", index, err)
				}
			}
		}

		// Apply the new schema
		if err := sqlitex.ExecuteScript(conn, sqlSchema, &sqlitex.ExecOptions{}); err != nil {
			return fmt.Errorf("applying schema: %w", err)
		}

		// Copy data from old table if it exists
		if hasOperationsTable {
			var ops []*v1.Operation
			batchInsert := func() error {
				if err := store.addInternal(conn, ops...); err != nil {
					return err
				}
				ops = nil
				return nil
			}

			if err := sqlitex.ExecuteTransient(conn, "SELECT operation FROM operations_old", &sqlitex.ExecOptions{
				ResultFunc: func(stmt *sqlite.Stmt) error {
					var op v1.Operation
					bytes := make([]byte, stmt.ColumnLen(0))
					n := stmt.ColumnBytes(0, bytes)
					bytes = bytes[:n]
					if err := proto.Unmarshal(bytes, &op); err != nil {
						return fmt.Errorf("unmarshal operation: %v", err)
					}

					ops = append(ops, &op)
					if len(ops) >= 512 {
						if err := batchInsert(); err != nil {
							return err
						}
					}
					return nil
				},
			}); err != nil {
				return fmt.Errorf("copying data from old table: %w", err)
			}

			if len(ops) > 0 {
				if err := batchInsert(); err != nil {
					return fmt.Errorf("copying data from old table: %w", err)
				}
			}

			if err := sqlitex.ExecuteTransient(conn, "DROP TABLE operations_old", nil); err != nil {
				return fmt.Errorf("dropping old table: %w", err)
			}
		}

		return nil
	}); err != nil {
		return fmt.Errorf("migrate sqlite schema from version %d to %d: %w", version, sqlSchemaVersion, err)
	}
	return nil
}
