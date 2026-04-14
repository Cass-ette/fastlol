package internal

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// Table prints a formatted table to stdout
func Table(headers []string, rows [][]string) {
	if len(headers) == 0 {
		return
	}

	// Use tabwriter with proper settings
	// MinWidth=1, TabWidth=0, Padding=1, PadChar=' ', Flags=tabwriter.FilterHTML
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.FilterHTML)

	// Compute max width per column (visual width, strip ANSI)
	maxWidths := make([]int, len(headers))
	for i, h := range headers {
		maxWidths[i] = visualWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(maxWidths) {
				w := visualWidth(cell)
				if w > maxWidths[i] {
					maxWidths[i] = w
				}
			}
		}
	}

	// Write header
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		pad := maxWidths[i] - visualWidth(h)
		if pad > 0 {
			fmt.Fprint(w, h+strings.Repeat(" ", pad))
		} else {
			fmt.Fprint(w, h)
		}
	}
	fmt.Fprintln(w, "")

	// Write separator
	for i, ww := range maxWidths {
		if i > 0 {
			fmt.Fprint(w, "  ")
		}
		fmt.Fprint(w, strings.Repeat("─", ww))
	}
	fmt.Fprintln(w, "")

	// Write rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(w, "  ")
			}
			stripped := stripANSI(cell)
			pad := maxWidths[i] - visualWidth(stripped)
			if pad > 0 {
				fmt.Fprint(w, stripped+strings.Repeat(" ", pad))
			} else {
				fmt.Fprint(w, stripped)
			}
		}
		fmt.Fprintln(w, "")
	}

	w.Flush()
}

// visualWidth returns display width of a string (handles ANSI)
func visualWidth(s string) int {
	// Strip ANSI first to get clean string
	clean := stripANSI(s)
	return displayWidth(clean)
}

// displayWidth returns display width of clean text (no ANSI)
func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if isWide(r) {
			width += 2
		} else {
			width += 1
		}
	}
	return width
}

// isWide returns true for CJK and other wide characters
func isWide(r rune) bool {
	// CJK Unified Ideographs
	if r >= 0x4e00 && r <= 0x9fff {
		return true
	}
	// CJK Unified Ideographs Extension A
	if r >= 0x3400 && r <= 0x4dbf {
		return true
	}
	// CJK Compatibility Ideographs / Strokes
	if r >= 0xf900 && r <= 0xfaff {
		return true
	}
	// CJK Radicals / Kangxi
	if r >= 0x2e80 && r <= 0x2eff {
		return true
	}
	// CJK Symbols and Punctuation
	if r >= 0x3000 && r <= 0x303f {
		return true
	}
	// Halfwidth and Fullwidth Forms
	if r >= 0xff00 && r <= 0xffef {
		return true
	}
	// Korean Hangul Syllables
	if r >= 0xac00 && r <= 0xd7af {
		return true
	}
	// Japanese Hiragana / Katakana
	if r >= 0x3040 && r <= 0x30ff {
		return true
	}
	return false
}

// stripANSI removes ANSI escape codes
func stripANSI(s string) string {
	result := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			// Skip the escape sequence [ .. m
			i += 2
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) && s[i] == 'm' {
				i++
			}
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
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
		return fmt.Sprintf("\033[32m%s\033[0m", pct)
	} else if rate <= 0.48 {
		return fmt.Sprintf("\033[31m%s\033[0m", pct)
	}
	return fmt.Sprintf("\033[33m%s\033[0m", pct)
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
