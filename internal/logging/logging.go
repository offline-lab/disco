package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns string representation of log level
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Config holds logging configuration
type Config struct {
	Level  LogLevel
	Format string
	File   string
}

var (
	currentLevel = INFO
	logger      *log.Logger
	jsonLog      *jsonLogger
)

// jsonLogger writes JSON-formatted logs
type jsonLogger struct {
	logger *log.Logger
}

// Log writes a JSON log entry
func (jl *jsonLogger) Log(level LogLevel, msg string, fields map[string]interface{}) {
	entry := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"message":   msg,
	}

	for k, v := range fields {
		entry[k] = v
	}

	data, _ := json.Marshal(entry)
	jl.logger.Println(string(data))
}

// Setup initializes the logger
func Setup(cfg Config) error {
	currentLevel = cfg.Level

	var output *os.File

	if cfg.File != "" {
		f, err := os.OpenFile(cfg.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}
		output = f
	} else {
		output = os.Stdout
	}

	if cfg.Format == "json" {
		jsonLog = &jsonLogger{
			logger: log.New(output, "", log.LstdFlags),
		}
	} else {
		flags := log.LstdFlags
		logger = log.New(output, "", flags)
	}

	return nil
}

// Debug logs a debug message
func Debug(msg string, fields map[string]interface{}) {
	if currentLevel > DEBUG {
		return
	}

	if jsonLog != nil {
		jsonLog.Log(DEBUG, msg, fields)
	} else {
		log.Printf("[DEBUG] %s", formatMessage(msg, fields))
	}
}

// Info logs an info message
func Info(msg string, fields map[string]interface{}) {
	if currentLevel > INFO {
		return
	}

	if jsonLog != nil {
		jsonLog.Log(INFO, msg, fields)
	} else {
		log.Printf("[INFO] %s", formatMessage(msg, fields))
	}
}

// Warn logs a warning message
func Warn(msg string, fields map[string]interface{}) {
	if currentLevel > WARN {
		return
	}

	if jsonLog != nil {
		jsonLog.Log(WARN, msg, fields)
	} else {
		log.Printf("[WARN] %s", formatMessage(msg, fields))
	}
}

// Error logs an error message
func Error(msg string, err error, fields map[string]interface{}) {
	if currentLevel > ERROR {
		return
	}

	allFields := make(map[string]interface{})
	for k, v := range fields {
		allFields[k] = v
	}

	if err != nil {
		allFields["error"] = err.Error()
	}

	if jsonLog != nil {
		jsonLog.Log(ERROR, msg, allFields)
	} else {
		log.Printf("[ERROR] %s %v", formatMessage(msg, allFields), err)
	}
}

// Fatal logs a fatal message and exits
func Fatal(msg string, err error) {
	if jsonLog != nil {
		fields := map[string]interface{}{}
		if err != nil {
			fields["error"] = err.Error()
		}
		jsonLog.Log(FATAL, msg, fields)
	} else {
		log.Printf("[FATAL] %s %v", msg, err)
	}
	os.Exit(1)
}

// formatMessage formats a log message with fields
func formatMessage(msg string, fields map[string]interface{}) string {
	if len(fields) == 0 {
		return msg
	}

	result := msg + " ["
	first := true

	for k, v := range fields {
		if !first {
			result += ", "
		}
		result += fmt.Sprintf("%s=%v", k, v)
		first = false
	}

	result += "]"
	return result
}
