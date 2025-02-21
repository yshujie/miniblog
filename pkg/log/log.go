package log

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (l *Logger) Info(format string, args ...interface{}) {
	// 实现日志记录逻辑
}
