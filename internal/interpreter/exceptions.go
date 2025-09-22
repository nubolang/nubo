package interpreter

import (
	"fmt"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/internal/exception"
)

func runExc(format string, args ...any) *exception.Expection {
	return exception.Create(format, args...).WithLevel(exception.LevelRuntime)
}

func wrapRunExc(err error, dg *debug.Debug) *exception.Expection {
	return exception.From(err, dg).WithLevel(exception.LevelRuntime)
}

func undefinedVariable(name string) *exception.Expection {
	return runExc("undefined variable '%s'", name)
}

func cannotOperateOn(typ any) *exception.Expection {
	return runExc("cannot operate on type '%s'", typ)
}

func expressionError(msg string) *exception.Expection {
	return runExc("failed to evaluate expression: %s", msg)
}

func typeError(format string, args ...any) *exception.Expection {
	return runExc(format, args...).WithLevel(exception.LevelType)
}

func typeMismatch(expected, got any) *exception.Expection {
	return runExc("type mismatch: expected '%s', got '%s'", expected, got).WithLevel(exception.LevelType)
}

func argError(excepted, got int) *exception.Expection {
	return runExc("expected %d arguments, got %d", excepted, got).WithLevel(exception.LevelType)
}

func importError(format string, args ...any) *exception.Expection {
	return runExc(fmt.Sprintf("failed to import - %s", fmt.Sprintf(format, args...)))
}
