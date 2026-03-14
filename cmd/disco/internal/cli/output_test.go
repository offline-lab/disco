package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestColorize(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		color Color
	}{
		{"red", "error", ColorRed},
		{"green", "success", ColorGreen},
		{"yellow", "warning", ColorYellow},
		{"blue", "info", ColorBlue},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Colorize(tt.text, tt.color)
			if IsTerminal(os.Stdout) {
				if !strings.Contains(result, string(tt.color)) {
					t.Errorf("Colorize() missing color code for %s", tt.name)
				}
				if !strings.Contains(result, tt.text) {
					t.Errorf("Colorize() missing original text for %s", tt.name)
				}
			} else {
				if result != tt.text {
					t.Errorf("Colorize() should return plain text when not terminal")
				}
			}
		})
	}
}

func TestColorizeStatus(t *testing.T) {
	tests := []struct {
		status string
		color  Color
	}{
		{"healthy", ColorGreen},
		{"stale", ColorYellow},
		{"lost", ColorRed},
		{"static", ColorBlue},
		{"unknown", ColorReset},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := ColorizeStatus(tt.status)
			if IsTerminal(os.Stdout) && tt.status != "unknown" {
				if !strings.Contains(result, string(tt.color)) {
					t.Errorf("ColorizeStatus(%s) expected color %s", tt.status, tt.color)
				}
			}
		})
	}
}

func TestTable(t *testing.T) {
	buf := &bytes.Buffer{}

	table := NewTable("HOSTNAME", "ADDRESS", "STATUS")
	table.AddRow("web1", "192.168.1.10", "healthy")
	table.AddRow("mail1", "192.168.1.11", "stale")

	_, _ = buf.WriteString("table output")

	if table == nil {
		t.Error("NewTable() returned nil")
	}
}

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		wantErr bool
	}{
		{
			name: "simple object",
			data: map[string]string{
				"hostname": "web1",
				"status":   "healthy",
			},
			wantErr: false,
		},
		{
			name:    "array",
			data:    []string{"web1", "mail1", "db1"},
			wantErr: false,
		},
		{
			name:    "nil",
			data:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := OutputJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("OutputJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input   string
		maxLen  int
		wantLen int
	}{
		{"short", 10, 5},
		{"exactly ten", 10, 10},
		{"this is a very long string", 10, 10},
		{"", 10, 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Truncate(tt.input, tt.maxLen)
			if len(result) > tt.maxLen {
				t.Errorf("Truncate() result too long: got %d chars, max %d", len(result), tt.maxLen)
			}
			if len(result) != tt.wantLen {
				t.Errorf("Truncate() wrong length: got %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

func TestJoinStrings(t *testing.T) {
	tests := []struct {
		strs []string
		sep  string
		want string
	}{
		{[]string{"a", "b", "c"}, ",", "a,b,c"},
		{[]string{"single"}, ",", "single"},
		{[]string{}, ",", ""},
		{[]string{"a", "b"}, " - ", "a - b"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			result := JoinStrings(tt.strs, tt.sep)
			if result != tt.want {
				t.Errorf("JoinStrings() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestIsTerminal(t *testing.T) {
	// Stdout in tests is typically not a terminal
	if IsTerminal(os.Stdout) {
		t.Log("Running in a terminal")
	} else {
		t.Log("Not running in a terminal (expected in test environment)")
	}

	// File should not be a terminal
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if IsTerminal(f) {
		t.Error("Regular file should not be detected as terminal")
	}
}
