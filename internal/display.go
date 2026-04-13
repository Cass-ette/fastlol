package internal

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Table prints a formatted table to stdout
func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	// Header
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	// Separator
	sep := make([]string, len(headers))
	for i, h := range headers {
		sep[i] = strings.Repeat("─", len(h)+2)
	}
	fmt.Fprintln(w, strings.Join(sep, "\t"))
	// Rows
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// Title prints a styled section title
func Title(text string) {
	fmt.Printf("\n\033[1;36m%s\033[0m\n\n", text)
}

// Warn prints a warning message
func Warn(text string) {
	fmt.Fprintf(os.Stderr, "\033[33m⚠ %s\033[0m\n", text)
}

// Error prints an error message
func Error(text string) {
	fmt.Fprintf(os.Stderr, "\033[31m✗ %s\033[0m\n", text)
}

// WinRateColor returns win rate with color based on value
func WinRateColor(rate float64) string {
	pct := fmt.Sprintf("%.1f%%", rate*100)
	if rate >= 0.52 {
		return fmt.Sprintf("\033[32m%s\033[0m", pct) // green
	} else if rate <= 0.48 {
		return fmt.Sprintf("\033[31m%s\033[0m", pct) // red
	}
	return fmt.Sprintf("\033[33m%s\033[0m", pct) // yellow
}

// WinRateColorPct formats a percentage value with color
func WinRateColorPct(pct float64) string {
	s := fmt.Sprintf("%.1f%%", pct)
	if pct >= 52 {
		return fmt.Sprintf("\033[32m%s\033[0m", s)
	} else if pct <= 48 {
		return fmt.Sprintf("\033[31m%s\033[0m", s)
	}
	return fmt.Sprintf("\033[33m%s\033[0m", s)
}
