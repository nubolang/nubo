package exception

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
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

	start := line - 4
	if start < 1 {
		start = 1
	}
	end := line + 4
	if end > len(lines) {
		end = len(lines)
	}

	width := len(fmt.Sprintf("%d", end))
	var out strings.Builder
	for i := start; i <= end; i++ {
		mark := " "
		if i == line {
			mark = color.New(color.FgRed).Sprint(">")
		}
		fmt.Fprintf(&out, "   %s %s %s\n", mark, color.New(color.FgHiBlack).Sprintf("%*d|", width, i), lines[i-1])
	}

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

	start := line - 4
	if start < 1 {
		start = 1
	}
	end := line + 4
	if end > len(lines) {
		end = len(lines)
	}

	width := len(fmt.Sprintf("%d", end))
	var out strings.Builder
	var linesOut strings.Builder
	for i := start; i <= end; i++ {
		mark := " "
		if i == line {
			mark = "&gt;"
			linesOut.WriteString(fmt.Sprintf("<span style=\"color:var(--color-red-400)\">%s %*d</span>\n", mark, width, i))
		} else {
			linesOut.WriteString(fmt.Sprintf("%s %*d\n", mark, width, i))
		}

		out.WriteString(lines[i-1])
		out.WriteRune('\n')
	}

	return out.String(), linesOut.String(), true
}
