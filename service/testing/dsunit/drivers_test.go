package dsunit

import (
	"database/sql"
	"testing"
)

func TestSQLDriversAndPureGoSQLiteAliases(t *testing.T) {
	registered := map[string]bool{}
	for _, name := range sql.Drivers() {
		registered[name] = true
	}
	for _, name := range []string{"mysql", "bigquery", "oracle", "sqlite", "sqlite3"} {
		if !registered[name] {
			t.Errorf("driver %s not registered", name)
		}
	}
	for _, name := range []string{"sqlite", "sqlite3"} {
		t.Run(name, func(t *testing.T) {
			db, err := sql.Open(name, ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			tx, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec("CREATE TABLE sample(id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
				t.Fatal(err)
			}
			if _, err = tx.Exec("INSERT INTO sample(name) VALUES(?)", "customer"); err != nil {
				t.Fatal(err)
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			var value string
			if err = db.QueryRow("SELECT name FROM sample WHERE id=?", 1).Scan(&value); err != nil || value != "customer" {
				t.Fatalf("query=%q err=%v", value, err)
			}
		})
	}
}
