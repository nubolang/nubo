package sql

import (
	"fmt"

	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

func NewDB(conn *SQLConn) (language.Object, error) {
	db := conn.DB
	driver := conn.Driver

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	configureDBConnection(db, driver)

	obj, err := dbStruct.NewInstance()
	if err != nil {
		return nil, err
	}

	inst := obj.GetPrototype().(*language.StructPrototype)
	inst.Unlock()
	defer inst.Lock()

	emptyList, _ := n.List(nil)
	inst.SetObject("query", n.Function(n.Describe(n.Arg("query", n.TString), n.Arg("args", n.TList, emptyList)).Returns(n.TTList(n.TDict)), fnQuery(conn)))
	inst.SetObject("exec", n.Function(n.Describe(n.Arg("query", n.TString), n.Arg("args", n.TList, emptyList)).Returns(n.TInt), fnExec(conn)))
	inst.SetObject("close", n.Function(n.EmptyDescribe(), func(a *n.Args) (any, error) {
		if db != nil {
			return nil, db.Close()
		}
		return nil, nil
	}))
	inst.SetObject("ping", n.Function(n.Describe().Returns(n.TBool), fnPing(conn)))

	return obj, nil
}

func fnQuery(conn *SQLConn) func(*n.Args) (any, error) {
	return func(args *n.Args) (any, error) {
		query := args.Name("query")
		queryArgs := args.Name("args")
		queryObjArgs := queryArgs.Value().([]language.Object)

		var queryRawArgs = make([]any, len(queryObjArgs))
		for i, rawArg := range queryObjArgs {
			raw, err := language.ToValue(rawArg, false)
			if err != nil {
				return nil, err
			}
			queryRawArgs[i] = raw
		}

		rows, err := conn.DB.Query(query.String(), queryRawArgs...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var result []language.Object
		cols, err := rows.Columns()
		if err != nil {
			return nil, err
		}

		keys := make([]language.Object, len(cols))
		for i, colName := range cols {
			keys[i] = n.String(colName, nil)
		}

		for rows.Next() {
			rowValues := make([]any, len(cols))
			pointers := make([]any, len(cols))
			for i := range rowValues {
				pointers[i] = &rowValues[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				return nil, err
			}
			values := make([]language.Object, len(cols))
			for i := range rowValues {
				value, err := language.FromValue(rowValues[i], false)
				if err != nil {
					return nil, err
				}
				values[i] = value
			}

			dict, err := language.NewDict(keys, values, n.TAny, n.TAny, nil)
			if err != nil {
				return nil, err
			}
			result = append(result, dict)
		}

		if err = rows.Err(); err != nil {
			return nil, err
		}

		return language.NewList(result, n.TDict, nil), nil
	}
}

func fnExec(conn *SQLConn) func(*n.Args) (any, error) {
	return func(args *n.Args) (any, error) {
		query := args.Name("query").String()
		queryArgs := args.Name("args").Value().([]language.Object)

		rawArgs := make([]any, len(queryArgs))
		for i, arg := range queryArgs {
			v, err := language.ToValue(arg, false)
			if err != nil {
				return nil, err
			}
			rawArgs[i] = v
		}

		res, err := conn.DB.Exec(query, rawArgs...)
		if err != nil {
			return nil, err
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}

		return n.Int(int(rowsAffected)), nil
	}
}

func fnPing(conn *SQLConn) func(args *n.Args) (any, error) {
	return func(args *n.Args) (any, error) {
		err := conn.DB.Ping()
		return n.Bool(err == nil), nil
	}
}
