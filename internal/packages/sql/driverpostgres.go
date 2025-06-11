package sql

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewPostgres() language.Object {
	driver := "postgres"

	return n.Function(n.Describe(
		n.Arg("user", n.TString),
		n.Arg("password", n.TString),
		n.Arg("dbname", n.TString),
		n.Arg("host", n.TString, n.String("127.0.0.1")),
		n.Arg("port", n.TInt, n.Int(5432)),
		n.Arg("sslmode", n.TString, n.String("disable")),
		n.Arg("timezone", n.TString, n.String("UTC")),
	).Returns(n.TStruct), func(args *n.Args) (any, error) {
		user := args.Name("user").String()
		password := args.Name("password").String()
		dbname := args.Name("dbname").String()
		host := args.Name("host").String()
		port := args.Name("port").Value().(int64)
		sslmode := args.Name("sslmode").String()
		timezone := args.Name("timezone").String()

		dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=%s sslmode=%s TimeZone=%s",
			user, password, host, port, dbname, sslmode, timezone)

		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, err
		}

		return NewSQLConn(driver, db, args.Get(0).Debug()), nil
	})
}
