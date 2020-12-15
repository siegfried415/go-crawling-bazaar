/*
 * Copyright 2018 The CovenantSQL Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	//"os" 

        //wyong, 20201213
	"github.com/sirupsen/logrus"
        "github.com/siegfried415/gdf-rebuild/utils/callinfo"

)

const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel logrus.Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

var (
	// PkgDebugLogFilter is the log filter
	// if package name exists and log level is more verbose, the log will be dropped
	PkgDebugLogFilter = map[string]logrus.Level{
		"metric": InfoLevel,
		"rpc":    InfoLevel,
	}

	//N->Y, wyong, 20201211
	// SimpleLog is the flag of simple log format
	// "Y" for true, "N" for false. defined in `go build`
	SimpleLog = "Y"
)

// Logger wraps logrus logger type.
type Logger logrus.Logger

// Fields defines the field map to pass to `WithFields`.
type Fields logrus.Fields

// CallerHook defines caller awareness hook for logrus.
type CallerHook struct {
	StackLevels []logrus.Level
}

func init() {
	AddHook(StandardCallerHook())
	SetFormatter(&ColoredTextFormatter{})
}

var (
	// std is the name of the standard logger in stdlib `log`
	// std = logrus.New()
)

// StandardLogger returns the standard logger.
func StandardLogger() *Logger {
	//wyong, 20200715 
	return (*Logger)(logrus.StandardLogger())

	/*
	//l := logrus.StandardLogger()
	l := logrus.New()
	l.SetLevel(logrus.InfoLevel)
	l.SetOutput(os.Stdout) 
	
	//wyong, 20201213
	l.SetFormatter(new(ColoredTextFormatter)) 

	return (*Logger)(l)
	*/
}

// Printf logs a message at level Info on the standard logger.
func (l *Logger) Printf(format string, args ...interface{}) {
	Printf(format, args...)
}

// NewCallerHook creates new CallerHook.
func NewCallerHook(stackLevels []logrus.Level) *CallerHook {
	return &CallerHook{
		StackLevels: stackLevels,
	}
}

// StandardCallerHook is a convenience initializer for LogrusStackHook{} with
// default args.
func StandardCallerHook() *CallerHook {
	// defined in `go build`
	if SimpleLog == "Y" {
		return NewCallerHook([]logrus.Level{})
	}

	return NewCallerHook(
		[]logrus.Level{
			logrus.PanicLevel, 
			logrus.FatalLevel, 
			logrus.ErrorLevel,

			//for debug purpose turn hook on warn,info and debug too, 
			//wyong, 20200612 
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		},
	)
}

// Fire defines hook event handler.
func (hook *CallerHook) Fire(entry *logrus.Entry) error {
	funcDesc, caller := hook.caller(entry)
	fields := strings.SplitN(funcDesc, ".", 2)
	if len(fields) > 0 {
		level, ok := PkgDebugLogFilter[fields[0]]
		if ok && entry.Level > level {
			nilLogger := logrus.New()
			nilLogger.Formatter = &NilFormatter{}
			entry.Logger = nilLogger
			return nil
		}
	}
	entry.Data["caller"] = caller

	//wyong, 20201211
	//var cMsg string 
	//cMsg := fmt.Sprintf(" \x1b[%dm%s\x1b[0m=", 36, entry.Message)
	//fmt.Sprintf(cMsg, "\033[1;34m %s \033[0m", entry.Message) 
	//entry.Message = cMsg 

	return nil
}

// Levels define hook applicable level.
func (hook *CallerHook) Levels() []logrus.Level {
	// defined in `go build`
	if SimpleLog == "Y" {
		return []logrus.Level{}
	}
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,

		//for debug purpose turn hook on warn,info and debug too, 
		//wyong, 20200612 
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
	}
}

func (hook *CallerHook) caller(entry *logrus.Entry) (relFuncName, caller string) {
	var skipFrames int
	if len(entry.Data) == 0 {
		// When WithField(s) is not used, we have 8 logrus frames to skip.
		skipFrames = 8
	} else {
		// When WithField(s) is used, we have 6 logrus frames to skip.
		skipFrames = 6
	}

	pcs := make([]uintptr, 12)
	stacks := make([]runtime.Frame, 0, 12)
	if runtime.Callers(skipFrames, pcs) > 0 {
		var foundCaller bool
		_frames := runtime.CallersFrames(pcs)
		for {
			f, more := _frames.Next()
			//log.Debugf("%s:%d %s\n", f.File, f.Line, f.Function)
			if !foundCaller && strings.HasSuffix(f.File, "logwrapper.go") && more {
				f, _ = _frames.Next()
				relFuncName = strings.TrimPrefix(f.Function, "github.com/siegfried415/gdf-rebuild/")
				caller = fmt.Sprintf("%s:%d %s", filepath.Base(f.File), f.Line, relFuncName)
				foundCaller = true
			}
			if foundCaller {
				stacks = append(stacks, f)
			}
			if !more {
				break
			}
		}
	}

	if len(stacks) > 0 {
		for _, level := range hook.StackLevels {
			if entry.Level == level {
				stacksStr := make([]string, 0, len(stacks))
				for i, s := range stacks {
					if s.Line > 0 {
						fName := strings.TrimPrefix(s.Function, "github.com/siegfried415/gdf-rebuild/")
						stackStr := fmt.Sprintf("#%d %s@%s:%d     ", i, fName, filepath.Base(s.File), s.Line)
						stacksStr = append(stacksStr, stackStr)
					}
				}
				entry.Data["stack"] = stacksStr
				break
			}
		}
	}

	return relFuncName, caller
}

// SetOutput sets the standard logger output.
func SetOutput(out io.Writer) {
	logrus.SetOutput(out)
}

// SetFormatter sets the standard logger formatter.
func SetFormatter(formatter logrus.Formatter) {
	logrus.SetFormatter(formatter)
}

// SetLevel sets the standard logger level.
func SetLevel(level logrus.Level) {
	logrus.SetLevel(level)
}

// GetLevel returns the standard logger level.
func GetLevel() logrus.Level {
	return logrus.GetLevel()
}

// ParseLevel parse the level string and returns the logger level.
func ParseLevel(lvl string) (logrus.Level, error) {
	return logrus.ParseLevel(lvl)
}

// SetStringLevel enforce current log level.
func SetStringLevel(lvl string, defaultLevel logrus.Level) {
	if lvl, err := ParseLevel(lvl); err != nil {
		SetLevel(defaultLevel)
	} else {
		SetLevel(lvl)
	}
}

// AddHook adds a hook to the standard logger hooks.
func AddHook(hook logrus.Hook) {
	logrus.AddHook(hook)
}


// WithError creates an entry from the standard logger and adds an error to it, using the value defined in ErrorKey as key.
func WithError(err error) *Entry {
	return WithField(logrus.ErrorKey, err)
}

// WithField creates an entry from the standard logger and adds a field to
// it. If you want multiple fields, use `WithFields`.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithField(key string, value interface{}) *Entry {
	return (*Entry)(logrus.WithField(key, value))
}

// WithFields creates an entry from the standard logger and adds multiple
// fields to it. This is simply a helper for `WithField`, invoking it
// once for each field.
//
// Note that it doesn't log until you call Debug, Print, Info, Warn, Fatal
// or Panic on the Entry it returns.
func WithFields(fields Fields) *Entry {
	return (*Entry)(logrus.WithFields(logrus.Fields(fields)))
}

// WithTime add time fields to log entry.
func WithTime(t time.Time) *Entry {
	return (*Entry)(logrus.WithTime(t))
}

// Debug logs a message at level Debug on the standard logger.
func Debug(args ...interface{}) {
	//logrus.Debug(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Debug(args...) 
}

// Print logs a message at level Info on the standard logger.
func Print(args ...interface{}) {
	//logrus.Print(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Print(args...) 
}

// Info logs a message at level Info on the standard logger.
func Info(args ...interface{}) {
	//logrus.Info(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Info(args...) 
}

// Warn logs a message at level Warn on the standard logger.
func Warn(args ...interface{}) {
	//logrus.Warn(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Warn(args...) 
}

// Warning logs a message at level Warn on the standard logger.
func Warning(args ...interface{}) {
	//logrus.Warning(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Warning(args...) 
}

// Error logs a message at level Error on the standard logger.
func Error(args ...interface{}) {
	//std.WithField("Func", getFuncPath(2)).Error(args...)
	//logrus.Error(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Error(args...) 
}

// Fatal logs a message at level Fatal on the standard logger.
func Fatal(args ...interface{}) {
	//std.WithField("Func", getFuncPath(2)).Fatal(args...)
	//logrus.Fatal(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Fatal(args...) 
}

// Panic logs a message at level Panic on the standard logger.
func Panic(args ...interface{}) {
	//std.WithField("Func", getFuncPath(2)).Panic(args...)
	//logrus.Panic(args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Panic(args...) 
}

// Debugf logs a message at level Debug on the standard logger.
func Debugf(format string, args ...interface{}) {
	//wyong, 20201213
	//logrus.Debugf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Debugf(format, args...) 
}

// Printf logs a message at level Info on the standard logger.
func Printf(format string, args ...interface{}) {
	//logrus.Printf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Printf(format, args...) 
}

// Infof logs a message at level Info on the standard logger.
func Infof(format string, args ...interface{}) {
	//wyong, 20201214
	//logrus.Infof(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Infof(format, args...) 
}

// Warnf logs a message at level Warn on the standard logger.
func Warnf(format string, args ...interface{}) {
	//wyong, 20201214
	//logrus.Warnf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Warnf(format, args...) 
}

// Warningf logs a message at level Warn on the standard logger.
func Warningf(format string, args ...interface{}) {
	//logrus.Warningf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Warningf(format, args...) 
}

// Errorf logs a message at level Error on the standard logger.
func Errorf(format string, args ...interface{}) {
	//logrus.Errorf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Errorf(format, args...) 
}

// Fatalf logs a message at level Fatal on the standard logger.
func Fatalf(format string, args ...interface{}) {
	//logrus.Fatalf(format, args...)
	entry := (*Entry)(logrus.WithFields(logrus.Fields{
                "Prefix":    callinfo.Prefix(),
                "Goid": callinfo.Goid(),
        }))
	entry.Fatalf(format, args...) 
}

// Panicf logs a message at level Panic on the standard logger.
func Panicf(format string, args ...interface{}) {
	logrus.Panicf(format, args...)
}

// Debugln logs a message at level Debug on the standard logger.
func Debugln(args ...interface{}) {
	logrus.Debugln(args...)
}

// Println logs a message at level Info on the standard logger.
func Println(args ...interface{}) {
	logrus.Println(args...)
}

// Infoln logs a message at level Info on the standard logger.
func Infoln(args ...interface{}) {
	logrus.Infoln(args...)
}

// Warnln logs a message at level Warn on the standard logger.
func Warnln(args ...interface{}) {
	logrus.Warnln(args...)
}

// Warningln logs a message at level Warn on the standard logger.
func Warningln(args ...interface{}) {
	logrus.Warningln(args...)
}

// Errorln logs a message at level Error on the standard logger.
func Errorln(args ...interface{}) {
	logrus.Errorln(args...)
}

// Fatalln logs a message at level Fatal on the standard logger.
func Fatalln(args ...interface{}) {
	logrus.Fatalln(args...)
}

// Panicln logs a message at level Panic on the standard logger.
func Panicln(args ...interface{}) {
	logrus.Panicln(args...)
}
