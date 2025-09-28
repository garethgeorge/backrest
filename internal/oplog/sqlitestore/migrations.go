package sqlitestore

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const sqlSchemaVersion = 6

var sqlSchema = fmt.Sprintf(`
PRAGMA user_version = %d;

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
CREATE INDEX operation_modno ON operations (modno);

CREATE TABLE operation_groups (
	ogid INTEGER PRIMARY KEY AUTOINCREMENT,
	instance_id STRING NOT NULL,
	original_instance_keyid STRING NOT NULL,
	repo_guid STRING NOT NULL,
	repo_id STRING NOT NULL,
	plan_id STRING NOT NULL
);
CREATE INDEX group_repo_instance ON operation_groups (repo_id, instance_id);
CREATE INDEX group_repo_guid ON operation_groups (repo_guid);
CREATE INDEX group_instance ON operation_groups (instance_id);
`, sqlSchemaVersion)

func migrateSystemInfoTable(store *SqliteStore, db *sql.DB) error {
	// Check if system_info table exists, if it does convert it to fields in kvstore
	var hasSystemInfoTable bool
	err := db.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='system_info'").Scan(&hasSystemInfoTable)
	if err != nil {
		return fmt.Errorf("checking for system_info table: %w", err)
	}

	if hasSystemInfoTable {
		var version int
		err := db.QueryRowContext(context.Background(), "SELECT version FROM system_info").Scan(&version)
		if err != nil {
			return fmt.Errorf("getting database schema version: %w", err)
		}

		if err := store.SetVersion(int64(version)); err != nil {
			return fmt.Errorf("setting database schema version: %w", err)
		}
	}

	return nil
}

func applySqliteMigrations(store *SqliteStore, db *sql.DB) error {
	if err := migrateSystemInfoTable(store, db); err != nil {
		return err
	}

	var version int
	err := db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version)
	if err != nil {
		return fmt.Errorf("getting database schema version: %w", err)
	}

	if version == sqlSchemaVersion {
		return nil
	}

	zap.S().Infof("applying oplog sqlite schema migration from storage schema %d to %d", version, sqlSchemaVersion)

	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("migrate sqlite schema: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Check if operations table exists and rename it
	var hasOperationsTable bool
	err = tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='operations'").Scan(&hasOperationsTable)
	if err != nil {
		return fmt.Errorf("checking for operations table: %w", err)
	}

	oldOpsFilePath := ""
	if hasOperationsTable {
		tempFile, err := os.CreateTemp("", "operations")
		if err != nil {
			return fmt.Errorf("creating temporary file: %w", err)
		}
		defer tempFile.Close()
		if err := dumpOperations(tx, tempFile); err != nil {
			return fmt.Errorf("dumping operations: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			return fmt.Errorf("closing temporary file: %w", err)
		}
		oldOpsFilePath = tempFile.Name()

		// drop all tables that we're about to replace
		drop_tables := []string{
			"operations",
			"operation_groups",
		}

		for _, table := range drop_tables {
			if _, err := tx.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE IF EXISTS %s", table)); err != nil {
				return fmt.Errorf("dropping table %s: %w", table, err)
			}
		}
	}

	// Apply the new schema
	for _, stmt := range strings.Split(sqlSchema, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(context.Background(), stmt); err != nil {
			return fmt.Errorf("applying schema: %w", err)
		}
	}

	// Copy data from old table if it exists
	if hasOperationsTable {
		oldOpsFile, err := os.Open(oldOpsFilePath)
		if err != nil {
			return fmt.Errorf("opening old operations file: %w", err)
		}
		defer oldOpsFile.Close()

		var ops []*v1.Operation
		batchInsert := func() error {
			if err := store.addInternal(tx, ops...); err != nil {
				return err
			}
			ops = nil
			return nil
		}

		if err := loadOperations(oldOpsFile, func(op *v1.Operation) error {
			ops = append(ops, op)
			if len(ops) >= 512 {
				if err := batchInsert(); err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("copying data from old table: %w", err)
		}

		if len(ops) > 0 {
			if err := batchInsert(); err != nil {
				return fmt.Errorf("copying data from old table: %w", err)
			}
		}

		if err := os.Remove(oldOpsFilePath); err != nil {
			return fmt.Errorf("removing old operations file: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migrate sqlite schema: commit: %w", err)
	}
	return nil
}

func dumpOperations(tx *sql.Tx, w io.Writer) error {
	writer := bufio.NewWriter(w)
	defer writer.Flush()

	rows, err := tx.QueryContext(context.Background(), "SELECT operation FROM operations")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		// Pre-allocate a buffer with 4 bytes for the length
		var b []byte
		if err := rows.Scan(&b); err != nil {
			return err
		}

		// Try deserializing to verify integrity
		var op v1.Operation
		if err := proto.Unmarshal(b, &op); err != nil {
			return fmt.Errorf("deserializing operation: %w", err)
		}

		var sizeBytesBuf [4]byte
		binary.LittleEndian.PutUint32(sizeBytesBuf[:], uint32(len(b)))
		if _, err := writer.Write(sizeBytesBuf[:]); err != nil {
			return err
		}
		if _, err := writer.Write(b); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}

func loadOperations(r io.Reader, forEach func(op *v1.Operation) error) error {
	reader := bufio.NewReader(r)
	for {
		var sizeBytesBuf [4]byte
		if _, err := io.ReadFull(reader, sizeBytesBuf[:]); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		size := binary.LittleEndian.Uint32(sizeBytesBuf[:])
		b := make([]byte, size)
		if n, err := io.ReadFull(reader, b); err != nil {
			return fmt.Errorf("read operation (%d bytes): %v", size, err)
		} else if n != int(size) {
			return fmt.Errorf("read operation (%d bytes): expected %d bytes, got %d bytes", size, size, n)
		}
		var op v1.Operation
		if err := proto.Unmarshal(b, &op); err != nil {
			return fmt.Errorf("unmarshal operation (%d bytes): %v", size, err)
		}
		if err := forEach(&op); err != nil {
			return err
		}
	}
	return nil
}
