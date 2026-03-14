package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type Color string

const (
	ColorReset  Color = "\033[0m"
	ColorRed    Color = "\033[31m"
	ColorGreen  Color = "\033[32m"
	ColorYellow Color = "\033[33m"
	ColorBlue   Color = "\033[34m"
)

func IsTerminal(file *os.File) bool {
	return term.IsTerminal(int(file.Fd()))
}

func Colorize(text string, color Color) string {
	if !IsTerminal(os.Stdout) {
		return text
	}
	return string(color) + text + string(ColorReset)
}

func ColorizeStatus(status string) string {
	switch status {
	case "healthy":
		return Colorize(status, ColorGreen)
	case "stale":
		return Colorize(status, ColorYellow)
	case "lost":
		return Colorize(status, ColorRed)
	case "static":
		return Colorize(status, ColorBlue)
	default:
		return status
	}
}

type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		headers: headers,
		widths:  widths,
	}
}

func (t *Table) AddRow(values ...string) {
	row := make([]string, len(t.headers))
	for i, v := range values {
		if i < len(t.headers) {
			row[i] = v
			if len(v) > t.widths[i] {
				t.widths[i] = len(v)
			}
		}
	}
	t.rows = append(t.rows, row)
}

func (t *Table) Print() {
	t.printHeader()
	t.printSeparator()
	for _, row := range t.rows {
		t.printRow(row...)
	}
}

func (t *Table) printHeader() {
	t.printRow(t.headers...)
}

func (t *Table) printSeparator() {
	totalWidth := 0
	for i, w := range t.widths {
		totalWidth += w + 2
		if i > 0 {
			totalWidth += 1
		}
	}
	fmt.Println(strings.Repeat("-", totalWidth))
}

func (t *Table) printRow(values ...string) {
	for i, v := range values {
		if i > 0 {
			fmt.Print("  ")
		}
		if i < len(t.widths) {
			fmt.Printf("%-*s", t.widths[i], v)
		} else {
			fmt.Print(v)
		}
	}
	fmt.Println()
}

func OutputJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-2] + ".."
}

func JoinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func PrintSuccess(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(Colorize(msg, ColorGreen))
}

func PrintError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(Colorize(msg, ColorRed))
}

func PrintWarning(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Print(Colorize(msg, ColorYellow))
}
