package exception

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	htm "html"

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
	statusCode int

	base error
	msg  string

	level Level

	debug *debug.Debug
	trace []*debug.Debug
}

func Create(format string, args ...any) *Expection {
	return &Expection{
		msg:        fmt.Sprintf(format, args...),
		level:      LevelFatal,
		trace:      make([]*debug.Debug, 0),
		statusCode: 500,
	}
}

func From(err error, dg *debug.Debug, otherwise ...string) *Expection {
	var exception *Expection
	if errors.As(err, &exception) {
		return exception.WithTrace(dg)
	}

	if err == nil {
		if len(otherwise) > 0 {
			return Create(otherwise[0]).WithDebug(dg)
		}
		return Create("unknown exception").WithDebug(dg)
	}

	msg := fmt.Sprintf("%v", err)
	if len(otherwise) > 0 {
		msg = strings.ReplaceAll(otherwise[0], "@err", err.Error())
	}

	created := Create(msg).WithBase(err)

	sourceDebug := unwrapDebug(err)
	if sourceDebug != nil {
		created.WithDebug(sourceDebug)
		if !sameDebugLine(sourceDebug, dg) {
			created.WithTrace(dg)
		}
		return created
	}

	return created.WithDebug(dg)
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

	if sameDebugLine(e.debug, trace) {
		return e
	}

	for _, tr := range e.trace {
		if sameDebugLine(tr, trace) {
			return e
		}
	}

	e.trace = append(e.trace, trace)
	return e
}

func (e *Expection) WithStatusCode(code int) *Expection {
	e.statusCode = code
	return e
}

func (e *Expection) Error() string {
	if e == nil {
		return "unknown exception"
	}

	var sb strings.Builder
	levelColor := color.New(color.Bold, color.FgRed)
	if e.level == LevelSyntax || e.level == LevelSemantic {
		levelColor = color.New(color.Bold, color.FgYellow)
	}

	msg := e.msg
	if msg == "" {
		msg = "unknown error"
	}

	sb.WriteString(levelColor.Sprintf("error[%s]", e.level))
	sb.WriteString(": ")
	sb.WriteString(color.New(color.FgRed).Sprintf("%s", msg))

	if e.debug != nil {
		fmt.Fprintf(&sb, "\n  source: %s:%d:%d", color.New(color.FgHiBlue).Sprint(e.debug.File), e.debug.Line, e.debug.Column)
		fmt.Fprintf(&sb, "\n  %s", color.New(color.Bold, color.FgHiBlack).Sprint("snippet:"))

		code, ok := showConsoleCodeError(e.debug.File, e.debug.Line)
		if ok {
			sb.WriteString(fmt.Sprintf("\n%s", code))
		}
	}

	frames := e.traceFrames()
	if len(frames) > 0 {
		fmt.Fprintf(&sb, "\n\n  %s", color.New(color.Bold, color.FgYellow).Sprint("stack:"))
		for idx, trace := range frames {
			fmt.Fprintf(&sb, "\n    #%02d %s:%d:%d", idx+1, trace.File, trace.Line, trace.Column)
		}
	}

	return sb.String()
}

func (e *Expection) GetMessage(html bool) string {
	if e == nil {
		return "unknown exception"
	}

	var sb strings.Builder

	if e.level == LevelSyntax || e.level == LevelSemantic {
		if html {
			fmt.Fprintf(&sb, "<span style=\"color:var(--color-yellow-400)\"><strong>%s</strong></span>", e.level)
		} else {
			sb.WriteString(color.New(color.Bold, color.FgYellow).Sprintf("%s", e.level))
		}
	} else {
		if html {
			fmt.Fprintf(&sb, "<span style=\"color:var(--color-red-400)\"><strong>%s</strong></span>", e.level)
		} else {
			sb.WriteString(color.New(color.Bold, color.FgRed).Sprintf("%s", e.level))
		}
	}

	if e.msg != "" {
		sb.WriteString(": ")
		if html {
			fmt.Fprintf(&sb, "<span style=\"color:var(--color-red-400)\">%s</span>", htm.EscapeString(e.msg))
		} else {
			sb.WriteString(color.New(color.FgRed).Sprintf("%s", e.msg))
		}
	}

	blue := color.New(color.FgHiBlue).SprintFunc()
	if e.debug != nil {
		sb.WriteRune(' ')
		if html {
			sb.WriteString("<span style=\"color:var(--color-teal-400)\">at</span>")
		} else {
			sb.WriteString(color.New(color.FgCyan).Sprint("at"))
		}
		sb.WriteRune(' ')
		if html {
			fmt.Fprintf(&sb, "<span style=\"color:var(--color-blue-400)\">%s</span>:<span style=\"color:var(--color-blue-400)\">%s</span>:<span style=\"color:var(--color-blue-400)\">%s</span>", blue(e.debug.File), blue(e.debug.Line), blue(e.debug.Column))
		} else {
			fmt.Fprintf(&sb, "%s:%s:%s", blue(e.debug.File), blue(e.debug.Line), blue(e.debug.Column))
		}
	}

	return sb.String()
}

func (e *Expection) HTML() *HtmlError {
	return &HtmlError{
		StatusCode: e.statusCode,
		err:        e,
	}
}

func (e *Expection) traceFrames() []*debug.Debug {
	if len(e.trace) == 0 {
		return nil
	}

	deduped := make([]*debug.Debug, 0, len(e.trace))
	seen := make(map[string]struct{}, len(e.trace)+1)

	if e.debug != nil {
		seen[debugKey(e.debug)] = struct{}{}
	}

	for _, trace := range e.trace {
		if trace == nil {
			continue
		}
		key := debugKey(trace)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		deduped = append(deduped, trace)
	}

	return deduped
}

func unwrapDebug(err error) *debug.Debug {
	_, _, dg := debug.Unwrap(err)
	if dg != nil {
		return dg
	}

	type debugProvider interface {
		GetDebug() *debug.Debug
	}

	var provider debugProvider
	if errors.As(err, &provider) {
		return provider.GetDebug()
	}

	return nil
}

func sameDebugLine(a, b *debug.Debug) bool {
	if a == nil || b == nil {
		return false
	}

	return filepath.Clean(a.File) == filepath.Clean(b.File) && a.Line == b.Line
}

func debugKey(d *debug.Debug) string {
	if d == nil {
		return ""
	}

	return filepath.Clean(d.File) + ":" + strconv.Itoa(d.Line)
}
