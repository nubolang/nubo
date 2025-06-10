package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/buger/goterm"
	"github.com/fatih/color"
)

// getMethodPadding calculates the padding needed for the method
func getMethodPadding(method string) (string, int) {
	var (
		maxLength = 6
		padding   = maxLength - len(method)
	)
	if len(method) > maxLength {
		padding = 0
	}
	return method, padding
}

// getColoredMethodName returns the method name with color based on its type
func getColoredMethodName(method string) string {
	switch strings.TrimSpace(method) {
	case "GET":
		return color.New(color.FgHiGreen).Sprint(method)
	case "POST":
		return color.New(color.FgHiBlue).Sprint(method)
	case "PUT":
		return color.New(color.FgHiCyan).Sprint(method)
	case "PATCH":
		return color.New(color.FgHiYellow).Sprint(method)
	case "DELETE":
		return color.New(color.FgHiRed).Sprint(method)
	default:
		return color.New(color.FgHiMagenta).Sprint(method)
	}
}

// doLog logs the server request
func doLog(start time.Time, method string, path string, cached bool) {
	method, padding := getMethodPadding(method)
	coloredMethod := getColoredMethodName(method)
	paddingDots := strings.Repeat(".", padding)

	coloredMethod = paddingDots + coloredMethod

	elapsedTime := time.Since(start).String()
	coloredTime := color.New(color.FgHiBlack).Sprint(elapsedTime)

	terminalWidth := goterm.Width()
	terminalWidth = terminalWidth - len(paddingDots+method) - len(path) - len(elapsedTime) - 5

	cachedText := "[cached]"
	cacheStatus := ""
	if cached {
		cacheStatus = color.New(color.FgHiBlue).Sprint(cachedText)
		terminalWidth -= len(cachedText)
	}

	if terminalWidth < 5 {
		terminalWidth = 5
	}

	dots := strings.Repeat(".", terminalWidth)

	fmt.Printf(" %s %s %s %s %s\n", coloredMethod, path, dots, cacheStatus, coloredTime)
}
