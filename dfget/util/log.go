/*
 * Copyright 1999-2018 Alibaba Group.
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

package util

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// DefaultLogTimeFormat defines the timestamp format.
const DefaultLogTimeFormat = "2006-01-02 15:04:05.000"

// CreateLogger creates a logger.
func CreateLogger(logPath string, logName string, logLevel string, sign string) *log.Logger {
	var (
		logger      *log.Logger
		logFilePath = path.Join(logPath, logName)
		level, err  = log.ParseLevel(logLevel)
	)
	if err != nil {
		level = log.InfoLevel
	}
	if err = os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
		var logFile *os.File
		if logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			logFile.Seek(0, 2)
			logger = log.New()
			logger.Out = logFile
			logger.Formatter = &DragonflyFormatter{TimestampFormat: DefaultLogTimeFormat, Sign: sign}
			logger.Level = level
			return logger
		}
	}
	panic(err)
}

// AddConsoleLog will add a ConsoleLog into logger's hooks.
// It will output logs to console when logger's outputting logs.
func AddConsoleLog(logger *log.Logger) {
	consoleLog := &log.Logger{
		Out:       os.Stdout,
		Formatter: logger.Formatter,
		Hooks:     make(log.LevelHooks),
		Level:     logger.Level,
	}
	logger.Hooks.Add(&ConsoleHook{logger: consoleLog, levels: log.AllLevels})
}

// ConsoleHook shows logs on console.
type ConsoleHook struct {
	logger *log.Logger
	levels []log.Level
}

// Fire implements Hook#Fire.
func (ch *ConsoleHook) Fire(entry *log.Entry) error {
	if ch.logger.Level >= entry.Level {
		switch entry.Level {
		case log.PanicLevel, log.FatalLevel:
			defer func() {
				recover()
			}()
			ch.logger.Panic(entry.Message)
		case log.ErrorLevel:
			ch.logger.Error(entry.Message)
		case log.WarnLevel:
			ch.logger.Warn(entry.Message)
		case log.InfoLevel:
			ch.logger.Info(entry.Message)
		case log.DebugLevel:
			ch.logger.Debug(entry.Message)
		}
	}
	return nil
}

// Levels implements Hook#Levels().
func (ch *ConsoleHook) Levels() []log.Level {
	return ch.levels
}

// DragonflyFormatter customizes the dragonfly log format.
type DragonflyFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string
	Sign            string
}

// Format implements Formatter#Format.
func (f *DragonflyFormatter) Format(entry *log.Entry) ([]byte, error) {
	b := &bytes.Buffer{}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = DefaultLogTimeFormat
	}
	f.appendValue(b, entry.Time.Format(timestampFormat), true)
	f.appendValue(b, strings.ToUpper(entry.Level.String()), true)
	if !IsEmptyStr(f.Sign) {
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
