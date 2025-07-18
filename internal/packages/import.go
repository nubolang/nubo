package packages

import (
	"strings"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/packages/http"
	"github.com/nubolang/nubo/internal/packages/io"
	"github.com/nubolang/nubo/internal/packages/json"
	"github.com/nubolang/nubo/internal/packages/layoutjs"
	"github.com/nubolang/nubo/internal/packages/log"
	"github.com/nubolang/nubo/internal/packages/math"
	"github.com/nubolang/nubo/internal/packages/process"
	"github.com/nubolang/nubo/internal/packages/random"
	"github.com/nubolang/nubo/internal/packages/sql"
	"github.com/nubolang/nubo/internal/packages/thread"
	"github.com/nubolang/nubo/internal/packages/time"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native/n"
)

const (
	BuiltInModulePrefix = "@std/"
)

var packageList = []string{"io", "math", "json", "log", "thread", "random", "process", "layoutjs", "sql", "time", "http"}

func ImportPackage(name string, dg *debug.Debug) (language.Object, bool) {
	if name == "@std" {
		pkg := n.NewPackage("@std", dg)
		for _, pkgName := range packageList {
			getPkg, ok := ImportPackage("@std/"+pkgName, dg)
			if !ok {
				continue
			}
			pkg.GetPrototype().SetObject(pkgName, getPkg)
		}
		return pkg, true
	}

	if !strings.HasPrefix(name, BuiltInModulePrefix) {
		return nil, false
	}

	name = strings.TrimPrefix(name, BuiltInModulePrefix)
	switch name {
	case "io":
		return io.NewIO(dg), true
	case "math":
		return math.NewMath(dg), true
	case "json":
		return json.NewJSON(dg), true
	case "log":
		return log.NewLog(dg), true
	case "thread":
		return thread.NewThread(dg), true
	case "random":
		return random.NewRandom(dg), true
	case "process":
		return process.NewProcess(dg), true
	case "layoutjs":
		return layoutjs.NewLayoutJS(dg), true
	case "sql":
		return sql.NewSQL(dg), true
	case "sql/driver/sqlite":
		return sql.NewSQLite(), true
	case "sql/driver/mysql":
		return sql.NewMySQL(), true
	case "sql/driver/postgres":
		return sql.NewPostgres(), true
	case "time":
		return time.NewTime(dg), true
	case "http":
		return http.NewHttp(dg), true
	}

	return nil, false
}
