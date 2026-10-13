package dsunit

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/sijms/go-ora/v2"
	_ "github.com/viant/bigquery"
	"modernc.org/sqlite"
)

func init() {
	// Keep legacy DSUnit fixtures using sqlite3 working without a C toolchain.
	// Embedding programs that already registered that name retain their driver.
	for _, name := range sql.Drivers() {
		if name == "sqlite3" {
			return
		}
	}
	sql.Register("sqlite3", &sqlite.Driver{})
}
