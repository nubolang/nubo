package packages

import (
	"strings"

	"github.com/nubolang/nubo/internal/packages/io"
	"github.com/nubolang/nubo/internal/packages/json"
	"github.com/nubolang/nubo/internal/packages/layoutjs"
	"github.com/nubolang/nubo/internal/packages/log"
	"github.com/nubolang/nubo/internal/packages/math"
	"github.com/nubolang/nubo/internal/packages/process"
	"github.com/nubolang/nubo/internal/packages/random"
	"github.com/nubolang/nubo/internal/packages/sql"
	"github.com/nubolang/nubo/internal/packages/thread"
	"github.com/nubolang/nubo/language"
)

const (
	BuiltInModulePrefix = "@std/"
)

func ImportPackage(name string) (language.Object, bool) {
	if !strings.HasPrefix(name, BuiltInModulePrefix) {
		return nil, false
	}

	name = strings.TrimPrefix(name, BuiltInModulePrefix)
	switch name {
	case "io":
		return io.NewIO(), true
	case "math":
		return math.NewMath(), true
	case "json":
		return json.NewJSON(), true
	case "log":
		return log.NewLog(), true
	case "thread":
		return thread.NewThread(), true
	case "random":
		return random.NewRandom(), true
	case "process":
		return process.NewProcess(), true
	case "layoutjs":
		return layoutjs.NewLayoutJS(), true
	case "sql":
		return sql.NewSQL(), true
	case "sql/driver/sqlite":
		return sql.NewSQLite(), true
	case "sql/driver/mysql":
		return sql.NewMySQL(), true
	case "sql/driver/postgres":
		return sql.NewPostgres(), true
	}

	return nil, false
}
