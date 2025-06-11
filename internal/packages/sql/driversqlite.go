package sql

import (
	"database/sql"
	"path/filepath"
	"strings"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
	_ "modernc.org/sqlite"
)

func NewSQLite() language.Object {
	driver := "sqlite"

	return n.Function(n.Describe(n.Arg("dsn", n.TString)).Returns(n.TStruct), func(args *n.Args) (any, error) {
		dsnObj := args.Name("dsn")
		dsn := sqliteDSN(dsnObj.String(), filepath.Dir(dsnObj.Debug().File))

		if !strings.Contains(dsn, "cache=") {
			if strings.Contains(dsn, "?") {
				dsn += "&cache=shared"
			} else {
				dsn += "?cache=shared"
			}
		}

		if !strings.Contains(dsn, "_journal=") {
			if strings.Contains(dsn, "?") {
				dsn += "&_journal=WAL"
			} else {
				dsn += "?_journal=WAL"
			}
		}

		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, err
		}

		return NewSQLConn(driver, db, dsnObj.Debug()), nil
	})
}

func sqliteDSN(dsn string, currentDir string) string {
	if strings.HasPrefix(dsn, ":memory:") || strings.HasPrefix(dsn, "file::memory:") {
		return dsn
	}

	path := strings.TrimPrefix(dsn, "file:")
	parts := strings.SplitN(path, "?", 2)
	filename := parts[0]
	opts := ""
	if len(parts) == 2 {
		opts = "?" + parts[1]
	}

	if !filepath.IsAbs(filename) {
		filename = filepath.Join(currentDir, filename)
	}

	return "file:" + filename + opts
}
