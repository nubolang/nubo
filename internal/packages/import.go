package packages

import (
	"strings"

	"github.com/nubolang/nubo/internal/packages/io"
	"github.com/nubolang/nubo/internal/packages/json"
	"github.com/nubolang/nubo/internal/packages/log"
	"github.com/nubolang/nubo/internal/packages/math"
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
	}

	return nil, false
}
