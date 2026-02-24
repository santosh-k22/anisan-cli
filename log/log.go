// Package log provides a thread-safe, structured logging infrastructure with filesystem-based persistence.
package log

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anisan-cli/anisan/filesystem"
	"github.com/anisan-cli/anisan/key"
	"github.com/anisan-cli/anisan/where"
	"github.com/samber/lo"
	logrus "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// enabled indicates the persistent logging state for the active application instance.
var enabled bool

// Setup initializes the logging subsystem, including file handles, formatting, and severity levels based on global configuration.
// Inoperative state: If logging is disabled, all subsequent log emissions are silently discarded.
func Setup() error {
	enabled = viper.GetBool(key.LogsWrite)
	if !enabled {
		return nil
	}

	dir := where.Logs()
	if dir == "" {
		return errors.New("log directory path is empty")
	}

	filename := fmt.Sprintf("%s.log", time.Now().Format("2006-01-02"))
	path := filepath.Join(dir, filename)

	if exists := lo.Must(filesystem.API().Exists(path)); !exists {
		lo.Must(filesystem.API().Create(path))
	}

	f, err := filesystem.API().OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	logrus.SetOutput(f)

	if viper.GetBool(key.LogsJson) {
		logrus.SetFormatter(&logrus.JSONFormatter{PrettyPrint: true})
	} else {
		logrus.SetFormatter(&logrus.TextFormatter{})
	}

	lvl := viper.GetString(key.LogsLevel)
	parsed, err := logrus.ParseLevel(lvl)
	if err != nil {
		parsed = logrus.InfoLevel
	}
	logrus.SetLevel(parsed)

	return nil
}

// Severity-Specific Log Emissions - these functions proxy messages to the configured backend when logging is enabled.

func Panic(args ...interface{}) {
	if enabled {
		logrus.Panic(args...)
	}
}
func Panicf(format string, args ...interface{}) {
	if enabled {
		logrus.Panicf(format, args...)
	}
}
func Fatal(args ...interface{}) {
	if enabled {
		logrus.Fatal(args...)
	}
}
func Fatalf(format string, args ...interface{}) {
	if enabled {
		logrus.Fatalf(format, args...)
	}
}
func Error(args ...interface{}) {
	if enabled {
		logrus.Error(args...)
	}
}
func Errorf(format string, args ...interface{}) {
	if enabled {
		logrus.Errorf(format, args...)
	}
}
func Warn(args ...interface{}) {
	if enabled {
		logrus.Warn(args...)
	}
}
func Warnf(format string, args ...interface{}) {
	if enabled {
		logrus.Warnf(format, args...)
	}
}
func Info(args ...interface{}) {
	if enabled {
		logrus.Info(args...)
	}
}
func Infof(format string, args ...interface{}) {
	if enabled {
		logrus.Infof(format, args...)
	}
}
func Debug(args ...interface{}) {
	if enabled {
		logrus.Debug(args...)
	}
}
func Debugf(format string, args ...interface{}) {
	if enabled {
		logrus.Debugf(format, args...)
	}
}
func Trace(args ...interface{}) {
	if enabled {
		logrus.Trace(args...)
	}
}
func Tracef(format string, args ...interface{}) {
	if enabled {
		logrus.Tracef(format, args...)
	}
}
