/*
 * Copyright 2022 https://github.com/siegfried415
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
	"bytes"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort" 
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

        "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	black  = 30 
	red    = 31
	green  = 32 
	yellow = 33
	blue   = 34
	magenta= 35 
	cyan   = 36
)

var ColorArray [7]int  
var baseTimestamp time.Time

func init() {
	ColorArray = [...]int{black, red, green, yellow, blue, magenta, cyan }
	baseTimestamp = time.Now()
}

type fieldKey string

// FieldMap allows customization of the key names for default fields.
type FieldMap map[fieldKey]string

func (f FieldMap) resolve(key fieldKey) string {
        if k, ok := f[key]; ok {
                return k
        }

        return string(key)
}

// TextFormatter formats logs into text
type ColoredTextFormatter struct {
	// Set to true to bypass checking for a TTY before outputting colors.
	ForceColors bool

	// Force disabling colors.
	DisableColors bool

	// Force quoting of all values
	ForceQuote bool

	// DisableQuote disables quoting for all values.
	// DisableQuote will have a lower priority than ForceQuote.
	// If both of them are set to true, quote will be forced on all values.
	DisableQuote bool

	// Override coloring based on CLICOLOR and CLICOLOR_FORCE. - https://bixense.com/clicolors/
	EnvironmentOverrideColors bool

	// Disable timestamp logging. useful when output is redirected to logging
	// system that already adds timestamps.
	DisableTimestamp bool

	// Enable logging the full timestamp when a TTY is attached instead of just
	// the time passed since beginning of execution.
	FullTimestamp bool

	// TimestampFormat to use for display when a full timestamp is printed
	TimestampFormat string

	// The fields are sorted by default for a consistent output. For applications
	// that log extremely frequently and don't use the JSON formatter this may not
	// be desired.
	DisableSorting bool

	// The keys sorting function, when uninitialized it uses sort.Strings.
	SortingFunc func([]string)

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool

	// PadLevelText Adds padding the level text so that all the levels output at the same length
	// PadLevelText is a superset of the DisableLevelTruncation option
	PadLevelText bool

	// QuoteEmptyFields will wrap empty fields in quotes if true
	QuoteEmptyFields bool

	// Whether the logger's out is to a terminal
	isTerminal bool

	// FieldMap allows users to customize the names of keys for default fields.
	// As an example:
	// formatter := &TextFormatter{
	//     FieldMap: FieldMap{
	//         FieldKeyTime:  "@timestamp",
	//         FieldKeyLevel: "@level",
	//         FieldKeyMsg:   "@message"}}
	FieldMap FieldMap

	// CallerPrettyfier can be set by the user to modify the content
	// of the function and file keys in the data when ReportCaller is
	// activated. If any of the returned value is the empty string the
	// corresponding key will be removed from fields.
	CallerPrettyfier func(*runtime.Frame) (function string, file string)

	terminalInitOnce sync.Once

	// The max length of the level text, generated dynamically on init
	levelTextMaxLength int
}


func (f *ColoredTextFormatter) checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (f *ColoredTextFormatter) init(entry *logrus.Entry) {
	if entry.Logger != nil {
		f.isTerminal = f.checkIfTerminal(entry.Logger.Out)
	}

	// Get the max length of the level text
	for _, level := range logrus.AllLevels {
		levelTextLength := utf8.RuneCount([]byte(level.String()))
		if levelTextLength > f.levelTextMaxLength {
			f.levelTextMaxLength = levelTextLength
		}
	}
}

func (f *ColoredTextFormatter) isColored() bool {
	/*
	isColored := f.ForceColors || (f.isTerminal && (runtime.GOOS != "windows"))

	if f.EnvironmentOverrideColors {
		switch force, ok := os.LookupEnv("CLICOLOR_FORCE"); {
		case ok && force != "0":
			isColored = true
		case ok && force == "0", os.Getenv("CLICOLOR") == "0":
			isColored = false
		}
	}

	return isColored && !f.DisableColors
	*/

	return true 

}


// Format renders a single log entry
func (f *ColoredTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(Fields)
	for k, v := range entry.Data {
		data[k] = v
	}

	//todo
	//logrus.prefixFieldClashes(data, f.FieldMap, entry.HasCaller())

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}

	var funcVal, fileVal string
	fixedKeys := make([]string, 0, 4+len(data))
	if !f.DisableTimestamp {
		fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyTime))
	}
	fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyLevel))
	if entry.Message != "" {
		fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyMsg))
	}

	//if entry.err != "" {
	//	fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyLogrusError))
	//}

	if entry.HasCaller() {
		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		} else {
			funcVal = entry.Caller.Function
			fileVal = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		}

		if funcVal != "" {
			fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyFunc))
		}
		if fileVal != "" {
			fixedKeys = append(fixedKeys, f.FieldMap.resolve(logrus.FieldKeyFile))
		}
	} 

	if !f.DisableSorting {
		if f.SortingFunc == nil {
			sort.Strings(keys)
			fixedKeys = append(fixedKeys, keys...)
		} else {
			if !f.isColored() {
				fixedKeys = append(fixedKeys, keys...)
				f.SortingFunc(fixedKeys)
			} else {
				f.SortingFunc(keys)
			}
		}
	} else {
		fixedKeys = append(fixedKeys, keys...)
	}

	fixedKeys = append(fixedKeys, keys...)


	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	f.terminalInitOnce.Do(func() { f.init(entry) })

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		//timestampFormat = defaultTimestampFormat
		timestampFormat = time.RFC3339
	}

	if f.isColored() {
		f.printColored(b, entry, keys, data, timestampFormat)
	}  else {

		for _, key := range fixedKeys {
			var value interface{}
			switch {
			case key == f.FieldMap.resolve(logrus.FieldKeyTime):
				value = entry.Time.Format(timestampFormat)
			case key == f.FieldMap.resolve(logrus.FieldKeyLevel):
				value = entry.Level.String()
			case key == f.FieldMap.resolve(logrus.FieldKeyMsg):
				value = entry.Message
			//case key == f.FieldMap.resolve(logrus.FieldKeyLogrusError):
			//	value = entry.err
			case key == f.FieldMap.resolve(logrus.FieldKeyFunc) && entry.HasCaller():
				value = funcVal
			case key == f.FieldMap.resolve(logrus.FieldKeyFile) && entry.HasCaller():
				value = fileVal
			default:
				value = data[key]
			}
			f.appendKeyValue(b, key, value)
		}
	} 

	//f.printColored(b, entry, keys, data, timestampFormat)

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func (f *ColoredTextFormatter) getMessageColor(goid int ) int {
	return ColorArray[goid % 7] 
}

func (f *ColoredTextFormatter) printColored(b *bytes.Buffer, entry *logrus.Entry, keys []string, data Fields, timestampFormat string) {

	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = magenta
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	levelText := strings.ToUpper(entry.Level.String())
	if !f.DisableLevelTruncation && !f.PadLevelText {
		levelText = levelText[0:4]
	}
	if f.PadLevelText {
		// Generates the format string used in the next line, for example "%-6s" or "%-7s".
		// Based on the max level text length.
		formatString := "%-" + strconv.Itoa(f.levelTextMaxLength) + "s"
		// Formats the level text by appending spaces up to the max length, for example:
		// 	- "INFO   "
		//	- "WARNING"
		levelText = fmt.Sprintf(formatString, levelText)
	}

	// Remove a single newline if it already exists in the message to keep
	// the behavior of logrus text_formatter the same as the stdlib log package
	entry.Message = strings.TrimSuffix(entry.Message, "\n")

	caller := ""
	if entry.HasCaller() {
		funcVal := fmt.Sprintf("%s()", entry.Caller.Function)
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)

		if f.CallerPrettyfier != nil {
			funcVal, fileVal = f.CallerPrettyfier(entry.Caller)
		}

		if fileVal == "" {
			caller = funcVal
		} else if funcVal == "" {
			caller = fileVal
		} else {
			caller = fileVal + " " + funcVal
		}
	}

	goid, ok := entry.Data["_gid"]
	if !ok {
		goid = 0  
	}

	messageColor := f.getMessageColor(goid.(int))
	prefix , ok := entry.Data["_prefix"]
	if !ok {
		prefix = ""
	}

	switch {
	case f.DisableTimestamp:
		//fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m%s %-44s ", levelColor, levelText, caller, entry.Message)
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m%s \x1b[%dm%-44s\x1b[0m ", levelColor, levelText, caller, messageColor, entry.Message)

	case !f.FullTimestamp:
		//fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%04d]%s %-44s ", levelColor, levelText, int(entry.Time.Sub(baseTimestamp)/time.Second), caller, entry.Message)
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%04d]%s \x1b[%dm%-44s\x1b[0m ", levelColor, levelText, /* int(entry.Time.Sub(baseTimestamp)/time.Second)*/ goid, /* caller */ prefix, messageColor, entry.Message)

	default:
		//fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%s]%s %-44s ", levelColor, levelText, entry.Time.Format(timestampFormat), caller, entry.Message)
		fmt.Fprintf(b, "\x1b[%dm%s\x1b[0m[%d]%s \x1b[%dm%-44s\x1b[0m ", levelColor, levelText, /*entry.Time.Format(timestampFormat)*/ goid, /* caller */ prefix, messageColor, entry.Message)
	}

	//uncomment following lines if you don't want Fields output.  
	for _, k := range keys {
		if strings.HasPrefix(k, "_" ) != true {
			v := data[k]
			fmt.Fprintf(b, " \x1b[%dm%s\x1b[0m=", /* levelColor */ messageColor, k)
			f.appendValue(b, v)
		}
	}
}

func (f *ColoredTextFormatter) needsQuoting(text string) bool {
	if f.ForceQuote {
		return true
	}
	if f.QuoteEmptyFields && len(text) == 0 {
		return true
	}
	if f.DisableQuote {
		return false
	}
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.' || ch == '_' || ch == '/' || ch == '@' || ch == '^' || ch == '+') {
			return true
		}
	}
	return false
}

func (f *ColoredTextFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	b.WriteByte('=')
	f.appendValue(b, value)
}

func (f *ColoredTextFormatter) appendValue(b *bytes.Buffer, value interface{}) {
	stringVal, ok := value.(string)
	if !ok {
		stringVal = fmt.Sprint(value)
	}

	if !f.needsQuoting(stringVal) {
		b.WriteString(stringVal)
	} else {
		b.WriteString(fmt.Sprintf("%q", stringVal))
	}
}
