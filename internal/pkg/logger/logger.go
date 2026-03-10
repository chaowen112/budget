package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Log is the global logger instance used throughout the application.
var Log *logrus.Logger

// Init initialises the global logrus logger.
//
// It writes to both stdout and a file under logDir (e.g. /var/log/budget/).
// If the directory or file cannot be created it falls back to stdout only.
func Init(level, logDir string) {
	Log = logrus.New()
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
	})

	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	Log.SetLevel(lvl)

	// Try to create log directory and open file.
	if logDir != "" {
		if mkErr := os.MkdirAll(logDir, 0o755); mkErr == nil {
			fp := filepath.Join(logDir, "budget.log")
			f, fErr := os.OpenFile(fp, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if fErr == nil {
				Log.SetOutput(io.MultiWriter(os.Stdout, f))
				return
			}
		}
	}

	// Fallback: stdout only.
	Log.SetOutput(os.Stdout)
}
