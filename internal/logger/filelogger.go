package logger

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileLogger wraps the file writer and provides logging functionality.
type FileLogger struct {
	writer io.WriteCloser
}

// InitFileLogger initializes file logging with size-based rotation.
// Logs are written to both stdout and the log file.
// Returns a FileLogger that can be used and closed properly.
func InitFileLogger(logFile string) *FileLogger {
	fileWriter := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10,   // MB
		MaxBackups: 5,    // Keep last 5 rotated files
		MaxAge:     30,   // Days
		Compress:   true, // Gzip old files
		LocalTime:  true, // Use local time in rotated filename
	}

	multiWriter := io.MultiWriter(os.Stdout, fileWriter)

	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return &FileLogger{
		writer: fileWriter,
	}
}

// Close closes the file logger (call in cleanup).
func (fl *FileLogger) Close() error {
	if fl.writer != nil {
		return fl.writer.Close()
	}
	return nil
}
