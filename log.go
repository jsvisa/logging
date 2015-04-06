//high level log wrapper, so it can output different log based on level
package logging

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
)

const (
	Ldate         = log.Ldate
	Llongfile     = log.Llongfile
	Lmicroseconds = log.Lmicroseconds
	Lshortfile    = log.Lshortfile
	LstdFlags     = log.LstdFlags
	Ltime         = log.Ltime
)

type (
	LogLevel int
	LogType  int
)

const (
	LOG_FATAL   = LogType(0x1)
	LOG_ERROR   = LogType(0x2)
	LOG_WARNING = LogType(0x4)
	LOG_INFO    = LogType(0x8)
	LOG_DEBUG   = LogType(0x10)
)

const (
	LOG_LEVEL_NONE  = LogLevel(0x0)
	LOG_LEVEL_FATAL = LOG_LEVEL_NONE | LogLevel(LOG_FATAL)
	LOG_LEVEL_ERROR = LOG_LEVEL_FATAL | LogLevel(LOG_ERROR)
	LOG_LEVEL_WARN  = LOG_LEVEL_ERROR | LogLevel(LOG_WARNING)
	LOG_LEVEL_INFO  = LOG_LEVEL_WARN | LogLevel(LOG_INFO)
	LOG_LEVEL_DEBUG = LOG_LEVEL_INFO | LogLevel(LOG_DEBUG)
	LOG_LEVEL_ALL   = LOG_LEVEL_DEBUG
)

const FORMAT_TIME_DAY string = "20060102"
const FORMAT_TIME_HOUR string = "2006010215"

const DEFAULT_BACKEND_NAME = "console"

type Logger map[string]*Backend

type Backend struct {
	_log *log.Logger

	level   LogLevel
	colored bool

	dailyRolling bool
	hourRolling  bool
	sizeRolling  bool
	fileName     string
	logSuffix    string
	rotateSize   int64
	fd           *os.File

	lock sync.Mutex
}

var _log Logger = NewSimpleLogger()

func init() {
	SetFlags(DEFAULT_BACKEND_NAME, Ldate|Ltime|Lshortfile)
}

func CrashLog(file string) {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println(err.Error())
	} else {
		syscall.Dup2(int(f.Fd()), 2)
	}
}

func AddBackend(name string, b *Backend) {
	_log[name] = b
}

func DeleteBackend(name string) {
	delete(_log, name)
}

func Info(v ...interface{}) {
	for _, l := range _log {
		l.Info(v...)
	}
}

func Infof(format string, v ...interface{}) {
	for _, l := range _log {
		l.Infof(format, v...)
	}
}

func Debug(v ...interface{}) {
	for _, l := range _log {
		l.Debug(v...)
	}
}

func Debugf(format string, v ...interface{}) {
	for _, l := range _log {
		l.Debugf(format, v...)
	}
}

func Warning(v ...interface{}) {
	for _, l := range _log {
		l.Warning(v...)
	}
}

func Warningf(format string, v ...interface{}) {
	for _, l := range _log {
		l.Warningf(format, v...)
	}
}

func Error(v ...interface{}) {
	for _, l := range _log {
		l.Error(v...)
	}
}

func Errorf(format string, v ...interface{}) {
	for _, l := range _log {
		l.Errorf(format, v...)
	}
}

func Fatal(v ...interface{}) {
	for _, l := range _log {
		l.Fatal(v...)
	}
}

func Fatalf(format string, v ...interface{}) {
	for _, l := range _log {
		l.Fatalf(format, v...)
	}
}

func SetLevel(name string, level LogLevel) {
	if b, ok := _log[name]; ok {
		b.level = level
	}
}

func GetLevel(name string) (LogLevel, error) {
	if b, ok := _log[name]; ok {
		return b.level, nil
	}

	return LOG_LEVEL_NONE, errors.New("Backend Not Found")
}

func SetOutput(name string, out io.Writer) {
	if b, ok := _log[name]; ok {
		b.SetOutput(out)
	}
}

func SetOutputByName(name string, path string) error {
	if b, ok := _log[name]; ok {
		return b.SetOutputByName(path)
	}
	return nil
}

func SetFlags(name string, flags int) {
	if b, ok := _log[name]; ok {
		b._log.SetFlags(flags)
	}
}

func SetColored(name string, colored bool) {
	if b, ok := _log[name]; ok {
		b.colored = colored
	}
}

func SetRotateByDay(name string) {
	if b, ok := _log[name]; ok {
		b.SetRotateByDay()
	}
}

func SetRotateByHour(name string) {
	if b, ok := _log[name]; ok {
		b.SetRotateByHour()
	}
}

func SetRotateBySize(name string, size int64) {
	if b, ok := _log[name]; ok {
		b.SetRotateBySize(size)
	}
}

func (b *Backend) SetLevelByString(level string) {
	b.level = StringToLogLevel(level)
}

func (b *Backend) SetRotateByDay() {
	b.dailyRolling = true
	b.logSuffix = genDayTime(time.Now())
}

func (b *Backend) SetRotateByHour() {
	b.hourRolling = true
	b.logSuffix = genHourTime(time.Now())
}

func (b *Backend) SetRotateBySize(size int64) {
	b.sizeRolling = true
	b.rotateSize = size
	b.logSuffix = genNextSeq(b.logSuffix)
}

func (b *Backend) rotate() error {
	b.lock.Lock()
	defer b.lock.Unlock()

	var suffix string
	switch {
	case b.dailyRolling:
		suffix = genDayTime(time.Now())
	case b.hourRolling:
		suffix = genHourTime(time.Now())
	case b.sizeRolling:
		suffix = genNextSeq(b.logSuffix)
	default:
		return nil
	}

	// Notice: if suffix is not equal to b.LogSuffix, then rotate
	// or current file size is bigger then b.rotateSize, then rotate
	if suffix != b.logSuffix {
		if b.sizeRolling {
			stat, err := b.fd.Stat()
			if err != nil {
				return err
			}
			if stat.Size() < b.rotateSize {
				return nil
			}
		}

		err := b.doRotate(suffix)
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *Backend) doRotate(suffix string) error {
	lastFileName := b.fileName + "." + b.logSuffix
	err := os.Rename(b.fileName, lastFileName)
	if err != nil {
		return err
	}

	// Notice: Not check error, is this ok?
	b.fd.Close()

	err = b.SetOutputByName(b.fileName)
	if err != nil {
		return err
	}

	b.logSuffix = suffix

	return nil
}

func (b *Backend) SetOutput(out io.Writer) {
	b._log = log.New(out, b._log.Prefix(), b._log.Flags())
}

func (b *Backend) SetOutputByName(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal(err)
	}

	b.SetOutput(f)

	b.fileName = path
	b.fd = f

	return err
}

func (b *Backend) Fatal(v ...interface{}) {
	b.log(LOG_FATAL, v...)
	os.Exit(-1)
}

func (b *Backend) Fatalf(format string, v ...interface{}) {
	b.logf(LOG_FATAL, format, v...)
	os.Exit(-1)
}

func (b *Backend) Error(v ...interface{}) {
	b.log(LOG_ERROR, v...)
}

func (b *Backend) Errorf(format string, v ...interface{}) {
	b.logf(LOG_ERROR, format, v...)
}

func (b *Backend) Warning(v ...interface{}) {
	b.log(LOG_WARNING, v...)
}

func (b *Backend) Warningf(format string, v ...interface{}) {
	b.logf(LOG_WARNING, format, v...)
}

func (b *Backend) Debug(v ...interface{}) {
	b.log(LOG_DEBUG, v...)
}

func (b *Backend) Debugf(format string, v ...interface{}) {
	b.logf(LOG_DEBUG, format, v...)
}

func (b *Backend) Info(v ...interface{}) {
	b.log(LOG_INFO, v...)
}

func (b *Backend) Infof(format string, v ...interface{}) {
	b.logf(LOG_INFO, format, v...)
}

func (b *Backend) log(t LogType, v ...interface{}) {
	if b.level|LogLevel(t) != b.level {
		return
	}

	err := b.rotate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return
	}

	v1 := make([]interface{}, len(v)+2)
	logStr, logColor := LogTypeToString(t)
	if b.colored {
		v1[0] = "\033" + logColor + "m[" + logStr + "]"
		copy(v1[1:], v)
		v1[len(v)+1] = "\033[0m"
	} else {
		v1[0] = "[" + logStr + "]"
		copy(v1[1:], v)
		v1[len(v)+1] = ""
	}

	s := fmt.Sprintln(v1...)
	b._log.Output(4, s)
}

func (b *Backend) logf(t LogType, format string, v ...interface{}) {
	if b.level|LogLevel(t) != b.level {
		return
	}

	err := b.rotate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		return
	}

	logStr, logColor := LogTypeToString(t)
	var s string
	if b.colored {
		s = "\033" + logColor + "m[" + logStr + "] " + fmt.Sprintf(format, v...) + "\033[0m"
	} else {
		s = "[" + logStr + "] " + fmt.Sprintf(format, v...)
	}
	b._log.Output(4, s)
}

func StringToLogLevel(level string) LogLevel {
	switch level {
	case "fatal":
		return LOG_LEVEL_FATAL
	case "error":
		return LOG_LEVEL_ERROR
	case "warn":
		return LOG_LEVEL_WARN
	case "warning":
		return LOG_LEVEL_WARN
	case "debug":
		return LOG_LEVEL_DEBUG
	case "info":
		return LOG_LEVEL_INFO
	default:
		return LOG_LEVEL_ALL
	}
}

func LogTypeToString(t LogType) (string, string) {
	switch t {
	case LOG_FATAL:
		return "fatal", "[0;31"
	case LOG_ERROR:
		return "error", "[0;31"
	case LOG_WARNING:
		return "warning", "[0;33"
	case LOG_DEBUG:
		return "debug", "[0;36"
	case LOG_INFO:
		return "info", "[0;37"
	default:
		return "unknown", "[0;37"
	}
}

func genDayTime(t time.Time) string {
	return t.Format(FORMAT_TIME_DAY)
}

func genHourTime(t time.Time) string {
	return t.Format(FORMAT_TIME_HOUR)
}

func genNextSeq(suffix string) string {
	if suffix == "" {
		return "0"
	}
	seq, _ := strconv.Atoi(suffix)
	return strconv.Itoa(seq + 1)
}

func NewSimpleLogger() Logger {
	return Logger{DEFAULT_BACKEND_NAME: NewSimpleBackend()}
}

func New() Logger {
	return make(Logger)
}

func NewSimpleBackend() *Backend {
	return NewBackend(os.Stdout, "", LOG_LEVEL_ALL, true)
}

func NewBackend(w io.Writer, prefix string, level LogLevel, colored bool) *Backend {
	return &Backend{_log: log.New(w, prefix, LstdFlags), level: level, colored: colored}
}
