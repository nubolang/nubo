package sql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewMySQL() language.Object {
	driver := "mysql"

	return n.Function(n.Describe(
		n.Arg("user", n.TString),
		n.Arg("password", n.TString),
		n.Arg("dbname", n.TString),
		n.Arg("host", n.TString, n.String("127.0.0.0")),
		n.Arg("port", n.TInt, n.Int(3306)),
		n.Arg("charset", n.TString, n.String("utf8mb4")),
		n.Arg("parseTime", n.TBool, n.Bool(true)),
	).Returns(n.TStruct), func(args *n.Args) (any, error) {
		user := args.Name("user").String()
		password := args.Name("password").String()
		dbname := args.Name("dbname").String()
		host := args.Name("host").String()
		port := args.Name("port").Value().(int64)
		charset := args.Name("charset").String()
		parseTime := args.Name("parseTime").Value().(bool)

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t",
			user, password, host, port, dbname, charset, parseTime)

		db, err := sql.Open(driver, dsn)
		if err != nil {
			return nil, err
		}

		return NewSQLConn(driver, db, args.Get(0).Debug()), nil
	})
}
