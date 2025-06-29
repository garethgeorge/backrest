package kvstore

import (
	"context"
	"fmt"
	"strings"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

const (
	escapeChar = "\\"
)

func escape(s string) string {
	return strings.NewReplacer(
		"%", escapeChar+"%",
		"_", escapeChar+"_",
	).Replace(s)
}

type sqliteKvStoreImpl struct {
	dbpool    *sqlitex.Pool
	tableName string
	indexName string

	// Cached queries
	createTableSQL   string
	createIndexSQL   string
	getSQL           string
	setSQL           string
	forEachAllSQL    string
	forEachPrefixSQL string
}

var _ KvStore = (*sqliteKvStoreImpl)(nil)

func NewSqliteKVStore(dbpool *sqlitex.Pool, basename string) (*sqliteKvStoreImpl, error) {
	store := &sqliteKvStoreImpl{
		dbpool:    dbpool,
		tableName: basename,
		indexName: basename + "_key_idx",

		// Build all SQL queries upfront
		createTableSQL: fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			value BLOB
		);`, basename),
		createIndexSQL:   fmt.Sprintf(`CREATE INDEX IF NOT EXISTS %s ON %s (key);`, basename+"_key_idx", basename),
		getSQL:           fmt.Sprintf("SELECT value FROM %s WHERE key = ?", basename),
		setSQL:           fmt.Sprintf("INSERT OR REPLACE INTO %s (key, value) VALUES (?, ?)", basename),
		forEachAllSQL:    fmt.Sprintf("SELECT key, value FROM %s ORDER BY key", basename),
		forEachPrefixSQL: fmt.Sprintf("SELECT key, value FROM %s WHERE key LIKE ? ESCAPE ? ORDER BY key", basename),
	}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *sqliteKvStoreImpl) init() error {
	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("init sqlite: %v", err)
	}
	defer s.dbpool.Put(conn)

	err = sqlitex.ExecuteTransient(conn, s.createTableSQL, nil)
	if err != nil {
		return fmt.Errorf("create %s table: %v", s.tableName, err)
	}

	err = sqlitex.ExecuteTransient(conn, s.createIndexSQL, nil)
	if err != nil {
		return fmt.Errorf("create %s index: %v", s.indexName, err)
	}

	return nil
}

func (s *sqliteKvStoreImpl) Get(key string) ([]byte, error) {
	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get from kvstore: %v", err)
	}
	defer s.dbpool.Put(conn)

	var value []byte
	err = sqlitex.ExecuteTransient(conn, s.getSQL, &sqlitex.ExecOptions{
		Args: []any{key},
		ResultFunc: func(stmt *sqlite.Stmt) error {
			value = make([]byte, stmt.ColumnLen(0))
			n := stmt.ColumnBytes(0, value)
			value = value[:n]
			return nil
		},
	})
	if err != nil {
		return nil, fmt.Errorf("get from kvstore: %v", err)
	}
	return value, nil
}

func (s *sqliteKvStoreImpl) Set(key string, value []byte) error {
	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("set to kvstore: %v", err)
	}
	defer s.dbpool.Put(conn)

	err = sqlitex.ExecuteTransient(conn, s.setSQL, &sqlitex.ExecOptions{
		Args: []any{key, value},
	})
	if err != nil {
		return fmt.Errorf("set to kvstore: %v", err)
	}
	return nil
}

func (s *sqliteKvStoreImpl) ForEach(prefix string, onRow func(key string, value []byte) error) error {
	conn, err := s.dbpool.Take(context.Background())
	if err != nil {
		return fmt.Errorf("foreach from kvstore: %v", err)
	}
	defer s.dbpool.Put(conn)

	var query string
	var args []any

	if len(prefix) == 0 {
		query = s.forEachAllSQL
	} else {
		query = s.forEachPrefixSQL
		args = []any{escape(prefix) + "%", escapeChar}
	}

	err = sqlitex.ExecuteTransient(conn, query, &sqlitex.ExecOptions{
		Args: args,
		ResultFunc: func(stmt *sqlite.Stmt) error {
			key := stmt.ColumnText(0)
			value := make([]byte, stmt.ColumnLen(1))
			n := stmt.ColumnBytes(1, value)
			value = value[:n]
			return onRow(key, value)
		},
	})
	if err != nil {
		return fmt.Errorf("foreach from kvstore: %v", err)
	}
	return nil
}
