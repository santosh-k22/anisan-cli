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
	"github.com/charmbracelet/log"
	"github.com/samber/lo"
	"github.com/spf13/viper"
)

var enabled bool
var logger *log.Logger

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

	logger = log.NewWithOptions(f, log.Options{
		ReportTimestamp: true,
	})

	if viper.GetBool(key.LogsJson) {
		logger.SetFormatter(log.JSONFormatter)
	} else {
		logger.SetFormatter(log.TextFormatter)
	}

	lvl := viper.GetString(key.LogsLevel)
	parsed, err := log.ParseLevel(lvl)
	if err != nil {
		parsed = log.InfoLevel
	}
	logger.SetLevel(parsed)

	return nil
}

func Panic(args ...interface{}) {
	if enabled && logger != nil {
		logger.Fatal(fmt.Sprint(args...))
	}
}
func Panicf(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Fatalf(format, args...)
	}
}
func Fatal(args ...interface{}) {
	if enabled && logger != nil {
		logger.Fatal(fmt.Sprint(args...))
	}
}
func Fatalf(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Fatalf(format, args...)
	}
}
func Error(args ...interface{}) {
	if enabled && logger != nil {
		logger.Error(fmt.Sprint(args...))
	}
}
func Errorf(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Errorf(format, args...)
	}
}
func Warn(args ...interface{}) {
	if enabled && logger != nil {
		logger.Warn(fmt.Sprint(args...))
	}
}
func Warnf(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Warnf(format, args...)
	}
}
func Info(args ...interface{}) {
	if enabled && logger != nil {
		logger.Info(fmt.Sprint(args...))
	}
}
func Infof(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Infof(format, args...)
	}
}
func Debug(args ...interface{}) {
	if enabled && logger != nil {
		logger.Debug(fmt.Sprint(args...))
	}
}
func Debugf(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Debugf(format, args...)
	}
}
func Trace(args ...interface{}) {
	if enabled && logger != nil {
		logger.Debug(fmt.Sprint(args...))
	}
}
func Tracef(format string, args ...interface{}) {
	if enabled && logger != nil {
		logger.Debugf(format, args...)
	}
}
