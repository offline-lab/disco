package logging

import (
	"os"
	"testing"
)

func TestSetup(t *testing.T) {
	cfg := Config{
		Level:  INFO,
		Format: "text",
		File:   "",
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
}

func TestSetup_WithFile(t *testing.T) {
	tmpFile := "/tmp/disco-test.log"
	defer os.Remove(tmpFile)

	cfg := Config{
		Level:  DEBUG,
		Format: "text",
		File:   tmpFile,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestSetup_WithJSON(t *testing.T) {
	cfg := Config{
		Level:  INFO,
		Format: "json",
		File:   "",
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Set level back to info for other tests
	if err := Setup(Config{Level: INFO, Format: "text", File: ""}); err != nil {
		t.Logf("Setup reset error: %v", err)
	}
}

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level  LogLevel
		expect string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
	}

	for _, tt := range tests {
		if tt.level.String() != tt.expect {
			t.Errorf("LogLevel.String() = %s, want %s", tt.level.String(), tt.expect)
		}
	}
}

func TestLogLevel_Unknown(t *testing.T) {
	var unknown LogLevel = 99
	str := unknown.String()
	if str != "UNKNOWN" {
		t.Errorf("Unknown LogLevel.String() = %s, want UNKNOWN", str)
	}
}

func TestDebug(t *testing.T) {
	Setup(Config{Level: DEBUG, Format: "text", File: ""})

	Debug("test message", map[string]interface{}{"key": "value"})

	// We can't easily capture output in tests without mocking
	// Just ensure it doesn't panic

	Setup(Config{Level: INFO, Format: "text", File: ""})
}

func TestInfo(t *testing.T) {
	Setup(Config{Level: INFO, Format: "text", File: ""})

	Info("test message", map[string]interface{}{"key": "value"})

	// Just ensure it doesn't panic
}

func TestWarn(t *testing.T) {
	Setup(Config{Level: WARN, Format: "text", File: ""})

	Warn("test message", map[string]interface{}{"key": "value"})

	// Just ensure it doesn't panic
}

func TestError(t *testing.T) {
	Setup(Config{Level: ERROR, Format: "text", File: ""})

	testErr := os.ErrNotExist
	Error("test message", testErr, map[string]interface{}{"key": "value"})

	// Just ensure it doesn't panic
}

func TestFatal(t *testing.T) {
	// Fatal should call os.Exit, so we can't test it directly
	// We'll just test that it exists and doesn't panic when Error is nil
	// Note: We can't actually run this test as it would exit the process

	// Test with nil error to avoid immediate exit
	// But we can't even do that without exiting

	// Skip this test as it calls os.Exit
	t.Skip("Fatal() calls os.Exit(1), cannot be tested directly")
}

func TestLogLevels_Filtering(t *testing.T) {
	Setup(Config{Level: ERROR, Format: "text", File: ""})

	// At ERROR level, DEBUG and INFO messages should not appear
	Debug("debug message", nil)
	Info("info message", nil)

	// ERROR should appear
	Error("error message", nil, map[string]interface{}{})

	// Reset to INFO for other tests
	Setup(Config{Level: INFO, Format: "text", File: ""})
}

func TestLogLevels_All(t *testing.T) {
	Setup(Config{Level: DEBUG, Format: "text", File: ""})

	// All messages should appear at DEBUG level
	Debug("debug message", nil)
	Info("info message", nil)
	Warn("warn message", nil)
	Error("error message", nil, nil)

	// Reset to INFO for other tests
	Setup(Config{Level: INFO, Format: "text", File: ""})
}

func TestFields(t *testing.T) {
	Setup(Config{Level: INFO, Format: "text", File: ""})

	fields := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	Info("message with fields", fields)

	// Just ensure it doesn't panic
}

func TestFields_Nil(t *testing.T) {
	Setup(Config{Level: INFO, Format: "text", File: ""})

	Info("message with nil fields", nil)

	// Just ensure it doesn't panic
}

func TestFields_Empty(t *testing.T) {
	Setup(Config{Level: INFO, Format: "text", File: ""})

	fields := map[string]interface{}{}
	Info("message with empty fields", fields)

	// Just ensure it doesn't panic
}

func TestJSONFormat(t *testing.T) {
	Setup(Config{Level: INFO, Format: "json", File: ""})

	Info("json test message", map[string]interface{}{"key": "value"})

	// We can't easily verify JSON output without capturing stdout
	// Just ensure it doesn't panic

	Setup(Config{Level: INFO, Format: "text", File: ""})
}

func TestTextFormat(t *testing.T) {
	Setup(Config{Level: INFO, Format: "text", File: ""})

	Info("text test message", map[string]interface{}{"key": "value"})

	// We can't easily verify text output without capturing stdout
	// Just ensure it doesn't panic
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		Level:  INFO,
		Format: "text",
		File:   "",
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
}

func TestConfig_FilePath(t *testing.T) {
	tmpFile := "/tmp/disco-test-file.log"
	defer os.Remove(tmpFile)

	cfg := Config{
		Level:  DEBUG,
		Format: "json",
		File:   tmpFile,
	}

	err := Setup(cfg)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

func TestMultipleSetup(t *testing.T) {
	cfg1 := Config{Level: DEBUG, Format: "text", File: ""}
	cfg2 := Config{Level: INFO, Format: "text", File: ""}

	err1 := Setup(cfg1)
	err2 := Setup(cfg2)

	if err1 != nil || err2 != nil {
		t.Errorf("Setup() errors = %v, %v", err1, err2)
	}

	Setup(Config{Level: INFO, Format: "text", File: ""})
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		fields   map[string]interface{}
		expected string
	}{
		{
			name:     "no fields",
			msg:      "test message",
			fields:   nil,
			expected: "test message",
		},
		{
			name:     "empty fields",
			msg:      "test message",
			fields:   map[string]interface{}{},
			expected: "test message",
		},
		{
			name:     "single field",
			msg:      "test",
			fields:   map[string]interface{}{"key": "value"},
			expected: "test [key=value]",
		},
		{
			name:     "multiple fields",
			msg:      "test",
			fields:   map[string]interface{}{"key1": "val1", "key2": 123},
			expected: "test [",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMessage(tt.msg, tt.fields)

			if tt.name == "multiple fields" {
				if !containsAll(result, "test [", "key1=", "key2=") {
					t.Errorf("formatMessage() = %s, want to contain all expected parts", result)
				}
			} else if result != tt.expected {
				t.Errorf("formatMessage() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
