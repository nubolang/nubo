package exception

import (
	"encoding/json"
	"runtime"

	"github.com/nubolang/nubo/version"
)

type JSONDebug struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	ColumnEnd int    `json:"column_end"`
}

type JSONErrorNubo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Docs      string `json:"docs"`
}

type JSONError struct {
	Nubo JSONErrorNubo `json:"_nubo"`

	StatusCode int `json:"status_code,omitempty"`

	Level   Level        `json:"level"`
	Message string       `json:"message"`
	Debug   *JSONDebug   `json:"debug,omitempty"`
	Stack   []*JSONDebug `json:"stack,omitempty"`
	Near    string       `json:"near,omitempty"`
}

func (e *Expection) JSON(pretty ...bool) ([]byte, error) {
	var p bool
	if len(pretty) > 0 {
		p = pretty[0]
	}

	je := &JSONError{
		Nubo: JSONErrorNubo{
			Version:   version.Version,
			GoVersion: runtime.Version(),
			Docs:      "https://nubo.mrtn.vip",
		},
		StatusCode: e.statusCode,
		Level:      e.level,
		Message:    e.msg,
		Stack:      make([]*JSONDebug, len(e.trace)),
	}

	if e.debug != nil {
		je.Debug = &JSONDebug{
			File:      e.debug.File,
			Line:      e.debug.Line,
			Column:    e.debug.Column,
			ColumnEnd: e.debug.ColumnEnd,
		}
	}

	for i, st := range e.trace {
		je.Stack[i] = &JSONDebug{
			File:      st.File,
			Line:      st.Line,
			Column:    st.Column,
			ColumnEnd: st.ColumnEnd,
		}
	}

	code, _, ok := showHtmlCodeError(e.debug.File, e.debug.Line)
	if ok {
		je.Near = code
	}

	if p {
		return json.MarshalIndent(je, "", "    ")
	}

	return json.Marshal(je)
}
