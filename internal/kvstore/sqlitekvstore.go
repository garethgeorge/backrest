package kvstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var ErrNotExist = errors.New("key does not exist")

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
	dbpool    *sql.DB
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

func NewSqliteKVStore(dbpool *sql.DB, basename string) (*sqliteKvStoreImpl, error) {
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
	_, err := s.dbpool.ExecContext(context.Background(), s.createTableSQL)
	if err != nil {
		return fmt.Errorf("create %s table: %v", s.tableName, err)
	}

	_, err = s.dbpool.ExecContext(context.Background(), s.createIndexSQL)
	if err != nil {
		return fmt.Errorf("create %s index: %v", s.indexName, err)
	}

	return nil
}

func (s *sqliteKvStoreImpl) Get(key string) ([]byte, error) {
	var value []byte
	err := s.dbpool.QueryRowContext(context.Background(), s.getSQL, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotExist
		}
		return nil, fmt.Errorf("get from kvstore: %v", err)
	}
	return value, nil
}

func (s *sqliteKvStoreImpl) Set(key string, value []byte) error {
	_, err := s.dbpool.ExecContext(context.Background(), s.setSQL, key, value)
	if err != nil {
		return fmt.Errorf("set to kvstore: %v", err)
	}
	return nil
}

func (s *sqliteKvStoreImpl) ForEach(prefix string, onRow func(key string, value []byte) error) error {
	var query string
	var args []any

	if len(prefix) == 0 {
		query = s.forEachAllSQL
	} else {
		query = s.forEachPrefixSQL
		args = []any{escape(prefix) + "%", escapeChar}
	}

	rows, err := s.dbpool.QueryContext(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("foreach from kvstore: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return fmt.Errorf("foreach from kvstore: %v", err)
		}
		if err := onRow(key, value); err != nil {
			return err
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("foreach from kvstore: %v", err)
	}

	return nil
}
