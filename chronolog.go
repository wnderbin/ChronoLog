package choronolog

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogLevel int

// defining logging levels
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelFatal
)

// string representation of logging levels
func (l LogLevel) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarning:
		return "WARNING"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// log entry
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type Config struct { // logger configuration, represents the logger settings
	FilePath              string        // path to log file
	MaxSize               int64         // maximum log size
	MaxAge                time.Duration // maximum storage time for compressed logs
	CompressOldLogs       bool          // do i need to compress old logs
	JSONFormat            bool          // whether to use JSON format
	TimestampFormat       string        // timestamp format
	RotationCheckInterval time.Duration // rotation check interval
}

type Logger struct {
	config      Config
	file        *os.File
	currentSize int64
	mu          sync.Mutex
	quitChan    chan struct{}
}

func New(config Config) (*Logger, error) {
	if config.MaxSize == 0 {
		config.MaxSize = 50 * 1024 * 1024 // default size = 50MB
	}
	if config.MaxAge == 0 {
		config.MaxAge = 7 * 24 * time.Hour // default max age = 1 week
	}
	if config.TimestampFormat == "" {
		config.TimestampFormat = time.RFC3339 // default timestamp format = "2006-01-02T15:04:05Z07:00"
	}
	if config.RotationCheckInterval == 0 {
		config.RotationCheckInterval = time.Minute // check rotation every minute
	}

	l := &Logger{
		config:   config,
		quitChan: make(chan struct{}),
	}

	if err := l.openFile(); err != nil {
		return nil, err
	}

	go l.rotationChecker()

	return l, nil
}

func (l *Logger) openFile() error { // creating | opening a log file
	if err := os.MkdirAll(filepath.Dir(l.config.FilePath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}
	file, err := os.OpenFile(l.config.FilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	l.file = file
	l.currentSize = info.Size()

	return nil
}

func (l *Logger) rotationChecker() { // checking the need for log rotation
	ticker := time.NewTicker(l.config.RotationCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.mu.Lock()
			if l.currentSize >= l.config.MaxSize {
				if err := l.rotate(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to rotate log file: %v\n", err)
				}
			}
			l.mu.Unlock()
		case <-l.quitChan:
			return
		}
	}
}

func (l *Logger) rotate() error { // performing log rotation
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close log file: %w", err)
	}

	backupName := l.config.FilePath + "." + time.Now().Format(time.RFC3339) // archive file
	if err := os.Rename(l.config.FilePath, backupName); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	if l.config.CompressOldLogs {
		go func() {
			if err := l.compressFile(backupName, backupName+".gz"); err != nil {
				fmt.Fprintf(os.Stderr, "failed to compress log file: %v\n", err)
				return
			}
			if err := os.Remove(backupName); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove old log file: %v\n", err)
			}
		}()
	}

	if err := l.openFile(); err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	go l.cleanupOldLogs()

	return nil
}

func (l *Logger) compressFile(src, dst string) error { // compressing too large logs
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	if _, err := io.Copy(gzWriter, srcFile); err != nil {
		return fmt.Errorf("failed to compress data %w", err)
	}

	return nil
}

func (l *Logger) cleanupOldLogs() { // self-cleaning of old logs
	files, err := filepath.Glob(l.config.FilePath + ".*.gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get log files: %v\n", err)
		return
	}

	cutoffTime := time.Now().Add(-l.config.MaxAge)

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove old log file: %v\n", err)
			}
		}
	}
}

func (l *Logger) write(level LogLevel, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
	}

	var logLine string
	if l.config.JSONFormat {
		jsonData, err := json.Marshal(entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal log entry: %v\n", err)
			return
		}
		logLine = string(jsonData) + "\n"
	} else {
		logLine = fmt.Sprintf("%s - [%s]: %s", entry.Timestamp, level.String(), message)
		logLine += "\n"
	}

	n, err := fmt.Fprintf(l.file, "%s", logLine)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to write to log file: %v\n", err)
		return
	}

	l.currentSize += int64(n)
}

func (l *Logger) Close() error {
	close(l.quitChan)
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Debug(msg string) {
	l.write(LevelDebug, msg)
}

func (l *Logger) Info(msg string) {
	l.write(LevelInfo, msg)
}

func (l *Logger) Warning(msg string) {
	l.write(LevelWarning, msg)
}

func (l *Logger) Error(msg string) {
	l.write(LevelError, msg)
}

func (l *Logger) Fatal(msg string) {
	l.write(LevelFatal, msg)
}
