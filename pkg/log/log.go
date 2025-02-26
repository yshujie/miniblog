package log

import (
	"fmt"
	"time"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Info(format string, args ...interface{}) {
	// 实现日志记录逻辑
	l.writeLog("info", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	// 实现日志记录逻辑
	l.writeLog("error", format, args...)
}

func (l *Logger) Fatal(format string, args ...interface{}) {
	// 实现日志记录逻辑
	l.writeLog("fatal", format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	// 实现日志记录逻辑
	l.writeLog("debug", format, args...)
}

func (l *Logger) Warn(format string, args ...interface{}) {
	// 实现日志记录逻辑
	l.writeLog("warn", format, args...)
}

func (l *Logger) writeLog(level string, format string, args ...interface{}) {
	if args == nil {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "[", level, "]", format)
	} else {
		fmt.Println(time.Now().Format("2006-01-02 15:04:05"), "[", level, "]", format, args)
	}

}
