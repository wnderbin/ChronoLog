package choronolog

import (
	"os"
	"testing"
)

func TestLogger(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testing")
	if err != nil {
		t.Fatal(err)
	}

	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	config := Config{
		FilePath:        tmpFile.Name(),
		MaxSize:         1024,
		CompressOldLogs: false,
	}

	log, err := New(config)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer log.Close()

	tests := []struct {
		name    string
		level   LogLevel
		message string
	}{
		{"debug", LevelDebug, "debug message"},
		{"info", LevelInfo, "info message"},
		{"warning", LevelWarning, "warning message"},
		{"error", LevelError, "error message"},
		{"fatal", LevelFatal, "fatal message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.write(tt.level, tt.message)
		})
	}
}
