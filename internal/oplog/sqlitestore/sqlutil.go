package sqlitestore

import (
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

// withSqliteTransaction should be used when the function only executes reads
func withSqliteTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc := sqlitex.Transaction(conn)
	err = f()
	endFunc(&err)
	return err
}

func withImmediateSqliteTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc, err := sqlitex.ImmediateTransaction(conn)
	if err != nil {
		return err
	}
	err = f()
	endFunc(&err)
	return err
}

func withExclusiveSqliteTransaction(conn *sqlite.Conn, f func() error) error {
	var err error
	endFunc, err := sqlitex.ExclusiveTransaction(conn)
	if err != nil {
		return err
	}
	err = f()
	endFunc(&err)
	return err
}
