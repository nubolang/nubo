package codehighlight

import (
	"fmt"
	"html"
	"strings"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/internal/lexer"
)

func highlightUnknown(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		return "<span style=\"color:#99a1af\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiBlack).Sprint(keyword)
	}
	return ""
}

func highlightKeyword(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		return "<span style=\"color:#ffd230\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiYellow, color.Bold).Sprint(keyword)
	}
	return ""
}

func highlightIdentifier(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		if keyword == "self" {
			return "<b style=\"color:#ff6467\">" + html.EscapeString(keyword) + "</b>"
		}
		return "<span style=\"color:#a684ff\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		if keyword == "self" {
			return color.New(color.FgHiRed, color.Bold).Sprint(keyword)
		}
		return color.New(color.FgMagenta).Sprint(keyword)
	}
	return ""
}

func highlightBracket(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		return "<span style=\"color:#f3f4f6\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiWhite).Sprint(keyword)
	}
	return ""
}

func highlightNumber(mode Mode, keyword string, base int) string {
	var prefix string

	switch base {
	case 16:
		prefix = "0x"
	case 8:
		prefix = "0o"
	case 2:
		prefix = "0b"
	}

	switch mode {
	case ModeHTML:
		return "<span style=\"color:#fe9a00\">" + html.EscapeString(prefix+keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiYellow).Sprint(prefix + keyword)
	}
	return ""
}

func highlightOperator(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		return "<span style=\"color:#d1d5dc\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiBlack).Sprint(keyword)
	}
	return ""
}

func highlightFunction(mode Mode, keyword string) string {
	if keyword == "type" {
		return highlightKeyword(mode, keyword)
	}

	switch mode {
	case ModeHTML:
		return "<span style=\"color:#8ec5ff\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiBlue).Sprint(keyword)
	}

	return ""
}

func highlightString(mode Mode, token *lexer.Token) string {
	quote := fmt.Sprint(token.Map["quote"])
	value := token.Value

	if quote != "`" {
		value = fmt.Sprintf("%s%s%s", quote, strings.ReplaceAll(token.Value, quote, "\\"+quote), quote)
	} else {
		value = fmt.Sprintf("%s%s%s", quote, token.Value, quote)
	}

	switch mode {
	case ModeHTML:
		return "<span style=\"color:#f0b100\">" + html.EscapeString(value) + "</span>"
	case ModeConsole:
		return color.New(color.FgYellow).Sprint(value)
	}
	return ""
}

func highlightNumeric(mode Mode, keyword string) string {
	switch mode {
	case ModeHTML:
		return "<span style=\"color:#fe9a00\">" + html.EscapeString(keyword) + "</span>"
	case ModeConsole:
		return color.New(color.FgHiYellow).Sprint(keyword)
	}
	return ""
}
