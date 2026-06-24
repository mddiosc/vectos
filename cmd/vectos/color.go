package main

import "os"

// ponytail: tiny ANSI helper, no color framework, no x/term dependency.
// Respects NO_COLOR (https://no-color.org) and only emits codes when stdout is
// a character device (TTY), so pipes, redirection and CI stay clean. Swap for
// lipgloss only if we ever need layout.
var colorEnabled = detectColor()

func detectColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
)

func colorize(code, s string) string {
	if !colorEnabled || s == "" {
		return s
	}
	return code + s + ansiReset
}

func bold(s string) string   { return colorize(ansiBold, s) }
func dim(s string) string    { return colorize(ansiDim, s) }
func red(s string) string    { return colorize(ansiRed, s) }
func green(s string) string  { return colorize(ansiGreen, s) }
func yellow(s string) string { return colorize(ansiYellow, s) }
func cyan(s string) string   { return colorize(ansiCyan, s) }
