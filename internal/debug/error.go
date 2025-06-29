package debug

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/version"
)

type DebugErr struct {
	err   error
	msg   string
	debug *Debug
}

func (de *DebugErr) Error() string {
	redBold := color.New(color.FgRed, color.Bold).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	blue := color.New(color.FgHiBlue).SprintFunc()

	var location string
	if de.debug != nil {
		location = fmt.Sprintf(": %s:%s:%s", blue(de.debug.File), blue(de.debug.Line), blue(de.debug.Column))
	}

	if de.err == nil {
		return fmt.Sprintf("%s%s", red(de.msg), location)
	}

	return fmt.Sprintf("%s %s%s", redBold(de.err.Error()+":"), red(de.msg), location)
}

func (de *DebugErr) Unwrap() error {
	return de.err
}

func NewError(base error, err string, debug ...*Debug) error {
	var dg *DebugErr
	if errors.As(base, &dg) {
		if dg.debug == nil && len(debug) > 0 {
			dg.debug = debug[0]
		}
		return dg
	}

	var d *Debug
	if len(debug) > 0 {
		d = debug[0]
	}

	return &DebugErr{
		err:   base,
		msg:   err,
		debug: d,
	}
}

// HtmlError returns the error message with debug information html
func (de DebugErr) HtmlError() *HtmlError {
	if de.debug == nil {
		return nil
	}

	htmlErr := NewHtmlError(&de)
	return htmlErr
}

// JSONError returns the error message with debug information json
func (de DebugErr) JSONError() (string, bool) {
	if de.debug == nil {
		return de.Error(), false
	}

	data := map[string]any{
		"version": version.Version,
		"message": de.Error(),
		"file":    de.debug.File + ":" + strconv.Itoa(de.debug.Line) + ":" + strconv.Itoa(de.debug.Column),
		"line":    de.debug.Line,
		"column":  de.debug.Column,
	}

	s, err := json.Marshal(&data)
	if err != nil {
		return de.Error(), false
	}

	return string(s), true
}
