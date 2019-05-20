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
	"path"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// DefaultLogTimeFormat defines the timestamp format.
const DefaultLogTimeFormat = "2006-01-02 15:04:05.000"

// InitLog initializes the file logger for process.
// logfile is used to stored generated log in local filesystem.
func InitLog(debug bool, logFilePath string, sign string) error {
	// set the log level
	logLevel := logrus.InfoLevel
	if debug {
		logLevel = logrus.DebugLevel
	}

	// create and log file
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return fmt.Errorf("failed to create log file %s: %v", logFilePath, err)
	}
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	logFile.Seek(0, 2)

	// create formatter for default log Logger
	formatter := &DragonflyFormatter{
		TimestampFormat: DefaultLogTimeFormat,
		Sign:            sign,
	}

	// set all details in log default logger
	logrus.SetLevel(logLevel)
	logrus.SetOutput(logFile)
	logrus.SetFormatter(formatter)
	return nil
}

// InitConsoleLog initializes console logger for process.
// console log will output the dfget client's log in console/terminal for
// debugging usage.
func InitConsoleLog(debug bool, sign string) {
	formatter := &DragonflyFormatter{
		TimestampFormat: DefaultLogTimeFormat,
		Sign:            sign,
	}

	logLevel := logrus.InfoLevel
	if debug {
		logLevel = logrus.DebugLevel
	}

	consoleLog := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     logLevel,
	}
	hook := &ConsoleHook{
		logger: consoleLog,
		levels: logrus.AllLevels,
	}
	logrus.AddHook(hook)
}

// CreateLogger creates a Logger.
func CreateLogger(logPath string, logName string, logLevel string, sign string) (*logrus.Logger, error) {
	// parse log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}

	// create log file path
	logFilePath := path.Join(logPath, logName)
	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		return nil, err
	}

	// open log file
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	logFile.Seek(0, 2)
	Logger := logrus.New()
	Logger.Out = logFile
	Logger.Formatter = &DragonflyFormatter{TimestampFormat: DefaultLogTimeFormat, Sign: sign}
	Logger.Level = level
	return Logger, nil
}

// AddConsoleLog will add a ConsoleLog into Logger's hooks.
// It will output logs to console when Logger's outputting logs.
func AddConsoleLog(Logger *logrus.Logger) {
	consoleLog := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: Logger.Formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     Logger.Level,
	}
	Logger.Hooks.Add(&ConsoleHook{logger: consoleLog, levels: logrus.AllLevels})
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

// ----------------------------------------------------------------------------

// IsDebug returns the log level is debug.
func IsDebug(level logrus.Level) bool {
	return level >= logrus.DebugLevel
}
