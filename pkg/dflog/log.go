/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dflog

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogConfig holds all configurable properties of log.
type LogConfig struct {
	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	// It defaults to 40 megabytes.
	MaxSize int `yaml:"maxSize" json:"maxSize"`
	// MaxBackups is the maximum number of old log files to retain.
	// The default value is 1.
	MaxBackups int `yaml:"maxBackups" json:"maxBackups"`

	// Path is the location of log file
	// The default value is logs/dfdaemon.log
	Path string `yaml:"path" json:"path"`
}

// DefaultLogTimeFormat defines the timestamp format.
const DefaultLogTimeFormat = "2006-01-02 15:04:05.000"

// Option is a functional configuration for the given logrus logger.
type Option func(l *logrus.Logger) error

// WithDebug sets the log level to debug.
func WithDebug(debug bool) Option {
	return func(l *logrus.Logger) error {
		if debug {
			l.SetLevel(logrus.DebugLevel)
		}
		return nil
	}
}

func getLumberjack(l *logrus.Logger) *lumberjack.Logger {
	if logger, ok := l.Out.(*lumberjack.Logger); ok {
		return logger
	}
	return nil
}

// WithLogFile sets the logger to output to the given file, with log rotation.
//
// If the given file is empty, nothing will be done.
//
// The maxSize is the maximum size in megabytes of the log file before it gets rotated.
// It defaults to 40 megabytes.
//
// The maxBackups is the maximum number of old log files to retain.
// The default value is 1.
func WithLogFile(f string, maxSize, maxBackups int) Option {
	return func(l *logrus.Logger) error {
		if f == "" {
			return nil
		}
		if maxSize <= 0 {
			maxSize = 40
		}
		if maxBackups <= 0 {
			maxBackups = 1
		}

		if logger := getLumberjack(l); logger == nil {
			l.SetOutput(&lumberjack.Logger{
				Filename:   f,
				MaxSize:    maxSize, // mb
				MaxBackups: maxBackups,
			})
		} else {
			logger.Filename = f
		}

		return nil
	}
}

// WithMaxSizeMB sets the max size of log files in MB. If the logger is not configured
// to use a log file, an error is returned.
func WithMaxSizeMB(max uint) Option {
	return func(l *logrus.Logger) error {
		if logger := getLumberjack(l); logger != nil {
			logger.MaxSize = int(max)
			return nil
		}
		return errors.Errorf("lumberjack is not configured")
	}
}

// WithConsole adds a hook to output logs to stdout.
func WithConsole() Option {
	return func(l *logrus.Logger) error {
		consoleLog := &logrus.Logger{
			Out:       os.Stdout,
			Formatter: l.Formatter,
			Hooks:     make(logrus.LevelHooks),
			Level:     l.Level,
		}
		hook := &ConsoleHook{
			logger: consoleLog,
			levels: logrus.AllLevels,
		}
		l.AddHook(hook)
		return nil
	}
}

// WithSign sets the sign in formatter.
func WithSign(sign string) Option {
	return func(l *logrus.Logger) error {
		l.Formatter = &DragonflyFormatter{
			TimestampFormat: DefaultLogTimeFormat,
			Sign:            sign,
		}
		return nil
	}
}

// Init initializes the logger with given options. If no option is provided,
// the logger's formatter will be set with an empty sign.
func Init(l *logrus.Logger, opts ...Option) error {
	opts = append([]Option{
		WithSign(""),
	}, opts...)
	for _, opt := range opts {
		if err := opt(l); err != nil {
			return err
		}
	}
	return nil
}

// ConsoleHook shows logs on console.
type ConsoleHook struct {
	logger *logrus.Logger
	levels []logrus.Level
}

// Fire implements Hook#Fire.
func (ch *ConsoleHook) Fire(entry *logrus.Entry) error {
	if ch.logger.Level >= entry.Level {
		switch entry.Level {
		case logrus.PanicLevel, logrus.FatalLevel:
			defer func() {
				recover()
			}()
			ch.logger.Panic(entry.Message)
		case logrus.ErrorLevel:
			ch.logger.Error(entry.Message)
		case logrus.WarnLevel:
			ch.logger.Warn(entry.Message)
		case logrus.InfoLevel:
			ch.logger.Info(entry.Message)
		case logrus.DebugLevel:
			ch.logger.Debug(entry.Message)
		}
	}
	return nil
}

// Levels implements Hook#Levels().
func (ch *ConsoleHook) Levels() []logrus.Level {
	return ch.levels
}

// DragonflyFormatter customizes the dragonfly log format.
type DragonflyFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string
	Sign            string
}

// Format implements Formatter#Format.
func (f *DragonflyFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultLogTimeFormat
	}
	f.appendValue(b, entry.Time.Format(timestampFormat), true)
	f.appendValue(b,
		fmt.Sprintf("%-4.4s", strings.ToUpper(entry.Level.String())),
		true)
	if f.Sign != "" {
		fmt.Fprintf(b, "sign:%s ", f.Sign)
	}
	b.WriteString(": ")
	if entry.Message != "" {
		f.appendValue(b, entry.Message, false)
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *DragonflyFormatter) appendValue(b *bytes.Buffer, value interface{}, withSpace bool) {
	switch value := value.(type) {
	case string:
		b.WriteString(value)
	case error:
		b.WriteString(value.Error())
	default:
		fmt.Fprint(b, value)
	}

	if withSpace {
		b.WriteByte(' ')
	}
}
