package exception

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/internal/debug"
)

type Level string

const (
	LevelFatal Level = "FatalError"

	LevelSyntax   Level = "SyntaxError"
	LevelSemantic Level = "SemanticError"

	LevelRuntime Level = "RuntimeError"
	LevelType    Level = "TypeError"
	LevelValue   Level = "ValueError"
)

type Expection struct {
	base error
	msg  string

	level Level

	debug *debug.Debug
	trace []*debug.Debug
}

func Create(format string, args ...any) *Expection {
	return &Expection{
		msg:   fmt.Sprintf(format, args...),
		level: LevelFatal,
		trace: make([]*debug.Debug, 0),
	}
}

func From(err error, dg *debug.Debug, otherwise ...string) *Expection {
	var exception *Expection
	if errors.As(err, &exception) {
		return exception.WithTrace(dg)
	}

	if len(otherwise) > 0 {
		otherwise[0] = strings.ReplaceAll(otherwise[0], "@err", err.Error())
		return Create(otherwise[0]).WithBase(err).WithDebug(dg)
	}

	return Create("%v", err).WithDebug(dg)
}

func (e *Expection) WithBase(err error) *Expection {
	e.base = err
	return e
}

func (e *Expection) WithLevel(level Level) *Expection {
	e.level = level
	return e
}

func (e *Expection) WithDebug(debug *debug.Debug) *Expection {
	if e.debug == nil {
		e.debug = debug
		return e
	}
	return e.WithTrace(debug)
}

func (e *Expection) WithTrace(trace *debug.Debug) *Expection {
	if trace == nil {
		return e
	}

	if len(e.trace) == 0 || e.trace[len(e.trace)-1] != trace {
		e.trace = append(e.trace, trace)
	}

	return e
}

func (e *Expection) Error() string {
	if e == nil {
		return "unknown exception"
	}

	var sb strings.Builder

	if e.level == LevelSyntax || e.level == LevelSemantic {
		sb.WriteString(color.New(color.Bold, color.FgYellow).Sprintf("%s", e.level))
	} else {
		sb.WriteString(color.New(color.Bold, color.FgRed).Sprintf("%s", e.level))
	}

	if e.msg != "" {
		sb.WriteString(": ")
		sb.WriteString(color.New(color.FgRed).Sprintf("%s", e.msg))
	}

	blue := color.New(color.FgHiBlue).SprintFunc()
	if e.debug != nil {
		sb.WriteRune(' ')
		sb.WriteString(color.New(color.FgCyan).Sprint("at"))
		sb.WriteRune(' ')

		if e.debug != nil {
			sb.WriteString(fmt.Sprintf("%s:%s:%s", blue(e.debug.File), blue(e.debug.Line), blue(e.debug.Column)))

			code, ok := showConsoleCodeError(e.debug.File, e.debug.Line)
			if ok {
				sb.WriteString(fmt.Sprintf("\n%s", code))
			}
		}
	}

	if len(e.trace) > 0 {
		sb.WriteRune('\n')
		sb.WriteString(color.New(color.FgYellow, color.Bold).Sprint("trace"))
		sb.WriteRune(':')
		for _, trace := range e.trace {
			sb.WriteString(fmt.Sprintf("\n\t%s:%s:%s", blue(trace.File), blue(trace.Line), blue(trace.Column)))
		}
	}

	return sb.String()
}
