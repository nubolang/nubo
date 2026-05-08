package exception

import (
	"bufio"
	"fmt"
	"html"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/nubolang/nubo/internal/codehighlight"
)

func showConsoleCodeError(path string, line int) (string, bool) {
	if path == "" || line <= 0 {
		return "", false
	}

	file, err := os.Open(path)
	if err != nil {
		return "", false
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if line > len(lines) {
		return "", false
	}

	start := max(1, line-3)
	end := min(line+3, len(lines))
	width := len(strconv.Itoa(end))

	linesS := strings.Join(lines, "\n")
	hl, err := codehighlight.NewHighlight(strings.NewReader(linesS))
	if err == nil {
		code, err := hl.HighlightConsole()
		if err == nil {
			lines = strings.Split(code, "\n")
		}
	}

	var out strings.Builder
	fmt.Fprintf(&out, "    %s\n", color.New(color.FgHiBlack).Sprint("|"))
	for i := start; i <= end; i++ {
		mark := "  "
		if i == line {
			mark = color.New(color.FgRed, color.Bold).Sprint(">>")
		}
		fmt.Fprintf(&out, "    %s %s %s\n", mark, color.New(color.FgHiBlack).Sprintf("%*d |", width, i), lines[i-1])
	}
	fmt.Fprintf(&out, "    %s", color.New(color.FgHiBlack).Sprint("|"))

	return out.String(), true
}

func showHtmlCodeError(path string, line int) (string, string, bool) {
	if path == "" || line <= 0 {
		return "", "", false
	}

	file, err := os.Open(path)
	if err != nil {
		return "", "", false
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if line > len(lines) {
		return "", "", false
	}

	start := max(1, line-4)
	end := min(line+4, len(lines))

	width := len(fmt.Sprintf("%d", end))

	linesS := strings.Join(lines, "\n")

	var shouldEscape = true
	hl, err := codehighlight.NewHighlight(strings.NewReader(linesS))
	if err == nil {
		code, err := hl.HighlightHTML()
		if err == nil {
			lines = strings.Split(code, "\n")
			shouldEscape = false
		}
	}

	var out strings.Builder
	var linesOut strings.Builder
	for i := start; i <= end; i++ {
		mark := " "
		if i == line {
			mark = "&gt;"
			fmt.Fprintf(&linesOut, "<span style=\"color:var(--color-red-400)\">%s %*d</span>\n", mark, width, i)
		} else {
			fmt.Fprintf(&linesOut, "%s %*d\n", mark, width, i)
		}

		if shouldEscape {
			out.WriteString(html.EscapeString(lines[i-1]))
		} else {
			out.WriteString(lines[i-1])
		}
		out.WriteRune('\n')
	}

	return out.String(), linesOut.String(), true
}
