package sqlitestore

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func withSqliteTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc := sqlitex.Transaction(conn)
	err = f()
	endFunc(&err)
	return err
}
