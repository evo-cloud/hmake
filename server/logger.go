package server

import (
	"fmt"
	"os"

	"github.com/Sirupsen/logrus"
)

// LogLevel defines the level of log record
type LogLevel int

// LogLevels
const (
	LogLvlFatal = iota
	LogLvlError
	LogLvlWarn
	LogLvlInfo
	LogLvlDebug
)

// LogDriver implements the log
type LogDriver interface {
	WriteLog(level LogLevel, ctx map[string]interface{}, args ...interface{}) error
	WriteLogf(level LogLevel, ctx map[string]interface{}, fmt string, args ...interface{}) error
}

type logger struct {
	ctx    map[string]interface{}
	driver LogDriver
}

// NewLogger creates a root logger
func NewLogger(cfgmgr ConfigManager) (Logger, error) {
	driver, err := newLogrusDriver(cfgmgr)
	if err != nil {
		return nil, err
	}
	return &logger{
		ctx:    make(map[string]interface{}),
		driver: driver,
	}, nil
}

func (l *logger) With(meta ...interface{}) Logger {
	newlog := &logger{
		ctx:    make(map[string]interface{}),
		driver: l.driver,
	}
	for k, v := range l.ctx {
		newlog.ctx[k] = v
	}
	num := len(meta)
	for i := 0; i < num; i++ {
		switch v := meta[i].(type) {
		case error:
			newlog.ctx["err"] = v
		case map[string]interface{}:
			for key, val := range v {
				newlog.ctx[key] = val
			}
		default:
			key := fmt.Sprintf("%v", v)
			i++
			if i >= num {
				panic("incorrect number of context metadata")
			}
			newlog.ctx[key] = meta[i]
		}
	}
	return newlog
}

func (l *logger) Fatal(args ...interface{}) {
	l.driver.WriteLog(LogLvlFatal, l.ctx, args)
	os.Exit(1)
}

func (l *logger) Error(args ...interface{}) {
	l.driver.WriteLog(LogLvlError, l.ctx, args)
}

func (l *logger) Warn(args ...interface{}) {
	l.driver.WriteLog(LogLvlWarn, l.ctx, args)
}

func (l *logger) Info(args ...interface{}) {
	l.driver.WriteLog(LogLvlInfo, l.ctx, args)
}

func (l *logger) Debug(args ...interface{}) {
	l.driver.WriteLog(LogLvlDebug, l.ctx, args)
}

func (l *logger) Fatalf(fmt string, args ...interface{}) {
	l.driver.WriteLogf(LogLvlFatal, l.ctx, fmt, args)
	os.Exit(1)
}

func (l *logger) Errorf(fmt string, args ...interface{}) {
	l.driver.WriteLogf(LogLvlError, l.ctx, fmt, args)
}

func (l *logger) Warnf(fmt string, args ...interface{}) {
	l.driver.WriteLogf(LogLvlWarn, l.ctx, fmt, args)
}

func (l *logger) Infof(fmt string, args ...interface{}) {
	l.driver.WriteLogf(LogLvlInfo, l.ctx, fmt, args)
}

func (l *logger) Debugf(fmt string, args ...interface{}) {
	l.driver.WriteLogf(LogLvlDebug, l.ctx, fmt, args)
}

type logrusConfig struct {
	Level string `json:"level"`
}

type logrusDriver struct {
	logger *logrus.Logger
}

func newLogrusDriver(cfgmgr ConfigManager) (*logrusDriver, error) {
	conf := &logrusConfig{Level: logrus.InfoLevel.String()}
	err := cfgmgr.Get("logging", conf)
	if err != nil {
		return nil, err
	}
	level, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		return nil, err
	}
	logger := logrus.New()
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Level = level
	return &logrusDriver{logger: logger}, nil
}

func (d *logrusDriver) WriteLog(level LogLevel, ctx map[string]interface{}, args ...interface{}) error {
	entry := d.logger.WithFields(logrus.Fields(ctx))
	switch level {
	case LogLvlFatal:
		entry.Fatal(args...)
	case LogLvlError:
		entry.Error(args...)
	case LogLvlWarn:
		entry.Warn(args...)
	case LogLvlInfo:
		entry.Info(args...)
	case LogLvlDebug:
		entry.Debug(args...)
	}
	return nil
}

func (d *logrusDriver) WriteLogf(level LogLevel, ctx map[string]interface{}, fmt string, args ...interface{}) error {
	entry := d.logger.WithFields(logrus.Fields(ctx))
	switch level {
	case LogLvlFatal:
		entry.Fatalf(fmt, args...)
	case LogLvlError:
		entry.Errorf(fmt, args...)
	case LogLvlWarn:
		entry.Warnf(fmt, args...)
	case LogLvlInfo:
		entry.Infof(fmt, args...)
	case LogLvlDebug:
		entry.Debugf(fmt, args...)
	}
	return nil
}
