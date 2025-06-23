package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/language"
	"github.com/nubolang/nubo/native"
	"github.com/nubolang/nubo/native/n"
)

var currentLogLevel = 1 // 0=DEBUG, 1=INFO, 2=WARN, 3=ERROR

var logLevels = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
}

func NewLog(dg *debug.Debug) language.Object {
	instance := n.NewPackage("log", dg)
	proto := instance.GetPrototype()

	proto.SetObject("debug", native.NewFunction(logFn("DEBUG")))
	proto.SetObject("info", native.NewFunction(logFn("INFO")))
	proto.SetObject("warn", native.NewFunction(logFn("WARN")))
	proto.SetObject("error", native.NewFunction(logFn("ERROR")))

	proto.SetObject("setLevel", native.NewTypedFunction(native.OneArg("level", language.TypeString), language.TypeVoid, setLogLevelFn))

	return instance
}

func logFn(level string) func(args []language.Object) (language.Object, error) {
	return func(args []language.Object) (language.Object, error) {
		if logLevels[level] < currentLogLevel {
			return nil, nil
		}

		msgs := make([]any, len(args))
		for i, a := range args {
			msgs[i] = a.Value()
		}

		msg := fmt.Sprintf("[%s] %s\n", level, fmt.Sprint(msgs...))
		var writer io.Writer
		if level == "ERROR" || level == "WARN" {
			writer = os.Stderr
		} else {
			writer = os.Stdout
		}

		writer.Write([]byte(msg))
		return nil, nil
	}
}

func setLogLevelFn(ctx native.FnCtx) (language.Object, error) {
	level, err := ctx.Get("level")
	if err != nil {
		return nil, err
	}

	arg := strings.ToUpper(level.Value().(string))
	if lvl, ok := logLevels[arg]; ok {
		currentLogLevel = lvl
		return nil, nil
	}

	return nil, fmt.Errorf("invalid log level: %s", arg)
}
