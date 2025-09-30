package exception

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"

	"github.com/nubolang/nubo/internal/debug"
	"github.com/nubolang/nubo/version"
)

//go:embed assets/*
var htmlFiles embed.FS

type HtmlError struct {
	StatusCode int
	err        *Expection
}

func (he *HtmlError) Error() string {
	return he.err.Error()
}

// Error returns the error message with HTML debug information
func (he *HtmlError) GetPage() string {
	tmpl, err := template.ParseFS(htmlFiles, "assets/error.html")
	if err != nil {
		return he.err.Error()
	}

	props, err := he.getTemplateProps()
	if err != nil {
		return he.err.Error()
	}

	var wr bytes.Buffer
	err = tmpl.Execute(&wr, props)

	if err != nil {
		return err.Error()
	}

	return wr.String()
}

func (he *HtmlError) getTemplateProps() (map[string]any, error) {
	cssFile, err := htmlFiles.Open("assets/error.css")
	if err != nil {
		return nil, he.err
	}
	defer cssFile.Close()

	css, err := io.ReadAll(cssFile)
	if err != nil {
		return nil, he.err
	}

	if he.err.debug == nil {
		return nil, he.err
	}

	code, lines, ok := showHtmlCodeError(he.err.debug.File, he.err.debug.Line)
	if !ok {
		return nil, he.err
	}

	return map[string]any{
		"Error": he.err.level,
		"Version": version.Version,
		"Style":   template.HTML("<style>" + string(css) + "</style>"),
		"Message": template.HTML(he.err.GetMessage(true)),
		"File":    he.err.debug.File + ":" + strconv.Itoa(he.err.debug.Line) + ":" + strconv.Itoa(he.err.debug.Column),
		"Line":    he.err.debug.Line,
		"Lines":   template.HTML(lines),
		"Code":    template.HTML(code),
		"Stack":   traceHtmlString(he.err.trace),
	}, nil
}

func (he *HtmlError) JSON() ([]byte, bool) {
	return nil, false
}

func traceHtmlString(trace []*debug.Debug) string {
	if trace == nil {
		return ""
	}

	var sb strings.Builder
	for _, trace := range trace {
		sb.WriteString(fmt.Sprintf("%s:%d:%d\n", trace.File, trace.Line, trace.Column))
	}

	return sb.String()
}
