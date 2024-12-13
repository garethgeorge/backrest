package sqlitestore

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "github.com/garethgeorge/backrest/gen/go/v1"
	"github.com/garethgeorge/backrest/internal/protoutil"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const sqlSchemaVersion = 2

var sqlSchema = fmt.Sprintf(`
PRAGMA user_version = %d;

CREATE TABLE IF NOT EXISTS system_info (version INTEGER NOT NULL);
INSERT INTO system_info (version)
SELECT 0 WHERE NOT EXISTS (SELECT 1 FROM system_info);

CREATE TABLE operations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ogid INTEGER NOT NULL,
	original_id INTEGER NOT NULL,
	modno INTEGER NOT NULL,
	flow_id INTEGER NOT NULL,
	start_time_ms INTEGER NOT NULL,
	status INTEGER NOT NULL,
	snapshot_id STRING NOT NULL,
	operation BLOB NOT NULL,
	FOREIGN KEY (ogid) REFERENCES operation_groups (ogid)
);
CREATE INDEX operation_snapshot_id ON operations (snapshot_id);
CREATE INDEX operation_ogid ON operations (ogid);
CREATE INDEX operation_flow_id ON operations (flow_id);
CREATE INDEX operations_start_time_ms ON operations (start_time_ms);

CREATE TABLE operation_groups (
	ogid INTEGER PRIMARY KEY AUTOINCREMENT,
	partition_id STRING NOT NULL,
	instance_id STRING NOT NULL,
	repo_id STRING NOT NULL,
	plan_id STRING NOT NULL
);
CREATE INDEX group_repo_instance ON operation_groups (repo_id, instance_id);
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

	return withSqliteTransaction(conn, func() error {
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
			if err := sqlitex.ExecuteTransient(conn, "ALTER TABLE operations RENAME TO operations_old", nil); err != nil {
				return fmt.Errorf("renaming operations table: %w", err)
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

			if err := batchInsert(); err != nil {
				return err
			}

			if err := sqlitex.ExecuteTransient(conn, "DROP TABLE operations_old", nil); err != nil {
				return fmt.Errorf("dropping old table: %w", err)
			}
		}

		return nil
	})
}

func writeOperation(w io.Writer, op *v1.Operation) error {
	bytes, err := proto.Marshal(op)
	if err != nil {
		return fmt.Errorf("marshal operation: %v", err)
	}
	buf := protowire.AppendFixed64(nil, uint64(len(bytes)))
	if _, err := w.Write(buf); err != nil {
		return fmt.Errorf("write operation: %v", err)
	}
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write operation: %v", err)
	}
	return nil
}

func readOperation(r io.Reader) (*v1.Operation, error) {
	var bytes [8]byte
	n, err := r.Read(bytes[:])
	if err != nil {
		return nil, fmt.Errorf("read operation: %v", err)
	} else if n != 8 {
		return nil, fmt.Errorf("read operation: unexpected number of bytes for length")
	}
	opLen := protowire.SizeFixed64()
	opBytes := make([]byte, opLen)
	n, err = r.Read(opBytes)
	if err != nil {
		return nil, fmt.Errorf("read operation: %v", err)
	} else if n != opLen {
		return nil, fmt.Errorf("read operation: unexpected number of bytes for operation")
	}
	op := &v1.Operation{}
	if err := proto.Unmarshal(opBytes, op); err != nil {
		return nil, fmt.Errorf("unmarshal operation: %v", err)
	}
	return op, nil
}

func createBackupFile(db *sqlite.Conn, backupFile string) error {
	var toClose []io.Closer
	var f io.WriteCloser
	var err error
	f, err = os.Create("." + backupFile + ".tmp")
	if err != nil {
		return fmt.Errorf("create backup file: %v", err)
	}
	toClose = append(toClose, f)

	if strings.HasSuffix(backupFile, ".gz") {
		gz := gzip.NewWriter(f)
		toClose = append(toClose, gz)
		f = gz
	}

	err = sqlitex.ExecuteTransient(db, "SELECT operation FROM operations", &sqlitex.ExecOptions{
		ResultFunc: func(stmt *sqlite.Stmt) error {
			var op v1.Operation
			bytes := make([]byte, stmt.ColumnLen(0))
			n := stmt.ColumnBytes(0, bytes)
			bytes = bytes[:n]
			if err := proto.Unmarshal(bytes, &op); err != nil {
				return fmt.Errorf("unmarshal operation bytes: %v", err)
			}
			return writeOperation(f, &op)
		},
	})

	for _, c := range toClose {
		if err := c.Close(); err != nil {
			return fmt.Errorf("close resource: %v", err)
		}
	}

	if err != nil {
		return err
	}

	if err := os.Rename("."+backupFile+".tmp", backupFile); err != nil {
		return fmt.Errorf("rename in-progress backup file: %v", err)
	}

	return nil
}

func restoreBackupFile(db *sqlite.Conn, backupFile string) error {
	var toClose []io.Closer
	defer func() {
		for _, c := range toClose {
			if err := c.Close(); err != nil {
				zap.L().Warn("error closing resource", zap.Error(err))
			}
		}
	}()

	f, err := os.Open(backupFile)
	if err != nil {
		return fmt.Errorf("open backup file: %v", err)
	}
	toClose = append(toClose, f)

	var reader io.Reader = f
	if strings.HasSuffix(backupFile, ".gz") {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("create gzip reader: %v", err)
		}
		toClose = append(toClose, gz)
		reader = gz
	}

	err = withSqliteTransaction(db, func() error {
		// Delete all tables (by dropping them)
		tablesToDelete := []string{}
		if err := sqlitex.ExecuteTransient(db, "SELECT name FROM sqlite_master WHERE type='table'", &sqlitex.ExecOptions{
			ResultFunc: func(stmt *sqlite.Stmt) error {
				tablesToDelete = append(tablesToDelete, stmt.ColumnText(0))
				return nil
			},
		}); err != nil {
			return fmt.Errorf("drop tables: %v", err)
		}

		for _, table := range tablesToDelete {
			if err := sqlitex.ExecuteTransient(db, fmt.Sprintf("DROP TABLE %s", table), &sqlitex.ExecOptions{}); err != nil {
				return fmt.Errorf("drop table %s: %v", table, err)
			}
		}

		// Re-create all tables
		if err := sqlitex.ExecuteScript(db, sqlSchema, &sqlitex.ExecOptions{}); err != nil {
			return fmt.Errorf("create tables: %v", err)
		}

		// Note that skipping deserializing would be more efficient, but this provides a much better integrity check.
		for {
			op, err := readOperation(reader)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return fmt.Errorf("read operation: %v", err)
			}

			if err := protoutil.ValidateOperation(op); err != nil {
				zap.S().Warnf("during operation log restore, operation %d failed validation: %v", op.Id, err)
			}

			if err := sqlitex.ExecuteTransient(db, "INSERT INTO operations (operation) VALUES (?)", &sqlitex.ExecOptions{
				Args: []any{&op},
			}); err != nil {
				return fmt.Errorf("insert operation %d: %v", op.Id, err)
			}
		}
		return nil
	})

	for _, c := range toClose {
		if err := c.Close(); err != nil {
			return err
		}
	}

	if !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func backupFilename(backupFilePrefix string, timestamp time.Time, schema int) string {
	return fmt.Sprintf("%s-%d-schema%04d.binpb.gz", backupFilePrefix, timestamp.UnixMilli(), schema)
}

func enforceBackupLimit(backupFilePrefix string, maxBackups int) error {
	files, err := os.ReadDir(filepath.Dir(backupFilePrefix))
	if err != nil {
		return fmt.Errorf("reading directory for backup files: %w", err)
	}

	type backupFile struct {
		name string
		time time.Time
	}
	var backups []backupFile

	prefix := strings.TrimSuffix(backupFilePrefix, "-") + "-"
	timestampRegex := regexp.MustCompile(`-(\d+)-schema\d{4}\.binpb\.gz$`)

	for _, file := range files {
		if !strings.HasPrefix(file.Name(), prefix) {
			continue
		}
		matches := timestampRegex.FindStringSubmatch(file.Name())
		if matches == nil || len(matches) != 2 {
			continue
		}

		timeMillis, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			continue
		}
		timestamp := time.UnixMilli(int64(timeMillis))
		backups = append(backups, backupFile{name: file.Name(), time: timestamp})
	}

	if len(backups) <= maxBackups {
		return nil
	}

	sort.Slice(backups, func(i, j int) bool {
		return backups[i].time.Before(backups[j].time)
	})

	for i := 0; i < len(backups)-maxBackups; i++ {
		path := filepath.Join(filepath.Dir(backupFilePrefix), backups[i].name)
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removing old backup %s: %w", path, err)
		}
		zap.L().Info("Removed backup file from old database migration", zap.String("file", path))
	}

	return nil
}
