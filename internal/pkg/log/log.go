package log

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/yshujie/miniblog/internal/pkg/known"
)

// 定义 Logger 接口
type Logger interface {
	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Panicw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})
	Sync()
}

// zapLogger，Logger 接口的实现类，内部使用 zap 库
type zapLogger struct {
	z *zap.Logger
}

// 确保 zapLogger 实现了 Logger 接口
var _ Logger = &zapLogger{}

var (
	mu sync.Mutex

	// 全局 logger
	std = NewLogger(NewOptions())
)

// Init 初始化全局 logger
func Init(opts *Options) {
	mu.Lock()
	defer mu.Unlock()

	std = NewLogger(opts)
}

// NewLogger 创建一个新的 logger
func NewLogger(opts *Options) *zapLogger {
	if opts == nil {
		opts = NewOptions()
	}

	// 设置日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(opts.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	// 创建默认的 encoder 配置
	encoderConfig := zap.NewProductionEncoderConfig()

	// 自定义 messageKey
	encoderConfig.MessageKey = "message"

	// 自定义 timeKey
	encoderConfig.TimeKey = "timestamp"

	// 自定义时间序列化函数
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	// 自定义 time.Duration 序列化函数，将时间序列化为经过的毫秒数（浮点数）
	encoderConfig.EncodeDuration = func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendFloat64(float64(d) / float64(time.Millisecond))
	}

	// 自定义日志级别序列化函数
	encoderConfig.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(l.String())
	}

	// 构建 zap.Logger 需要的配置
	cfg := zap.Config{
		// 是否在日志中显示调用日志所在的文件和行号，例如：`"caller":"miniblog/miniblog.go:75"`
		DisableCaller: opts.DisableCaller,
		// 是否禁止在 panic 及以上级别打印堆栈信息
		DisableStacktrace: opts.DisableStacktrace,
		// 指定日志级别
		Level: zap.NewAtomicLevelAt(zapLevel),
		// 指定日志显示格式
		// 指定日志显示格式，可选值：console, json
		Encoding:      opts.Format,
		EncoderConfig: encoderConfig,

		// 指定日志输出位置
		OutputPaths: opts.OutputPaths,

		// 设置 zap 内部错误输出位置
		ErrorOutputPaths: []string{"stderr"},
	}

	// 根据配置创建 zap.Logger
	z, err := cfg.Build(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic(err)
	}

	// 创建 zapLogger 实例
	logger := &zapLogger{z: z}

	// 把标准库的 log.Logger 的 info 级别的输出重定向到 zap.Logger
	zap.RedirectStdLog(z)

	return logger
}

func Sync() {
	std.Sync()
}

func (l *zapLogger) Sync() {
	l.z.Sync()
}

// Debugw 记录 debug 级别的日志
func Debugw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Debugw(msg, keysAndValues...)
}

func (l *zapLogger) Debugw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Debugw(msg, keysAndValues...)
}

// Infow 记录 info 级别的日志
func Infow(msg string, keysAndValues ...interface{}) {
	fmt.Printf("Logging info message: %s with values: %v\n", msg, keysAndValues)
	std.z.Sugar().Infow(msg, keysAndValues...)
}

func (l *zapLogger) Infow(msg string, keysAndValues ...interface{}) {
	fmt.Printf("Logging info message with logger: %s with values: %v\n", msg, keysAndValues)
	l.z.Sugar().Infow(msg, keysAndValues...)
}

// Warnw 记录 warn 级别的日志
func Warnw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Warnw(msg, keysAndValues...)
}

func (l *zapLogger) Warnw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Warnw(msg, keysAndValues...)
}

// Errorw 记录 error 级别的日志
func Errorw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Errorw(msg, keysAndValues...)
}

func (l *zapLogger) Errorw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Errorw(msg, keysAndValues...)
}

// Panicw 记录 panic 级别的日志
func Panicw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Panicw(msg, keysAndValues...)
}

func (l *zapLogger) Panicw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Panicw(msg, keysAndValues...)
}

// Fatalw 记录 fatal 级别的日志
func Fatalw(msg string, keysAndValues ...interface{}) {
	std.z.Sugar().Fatalw(msg, keysAndValues...)
}

func (l *zapLogger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.z.Sugar().Fatalw(msg, keysAndValues...)
}

// C 解析传入的 context， 尝试提取关注的键，并添加到 zap.Logger 中
func C(ctx context.Context) *zapLogger {
	fmt.Printf("Creating logger from context: %v\n", ctx)
	return std.C(ctx)
}

func (l *zapLogger) C(ctx context.Context) *zapLogger {
	fmt.Printf("Creating logger from context with logger: %v\n", ctx)
	lc := l.clone()

	// 添加请求ID
	if requestID := ctx.Value(known.XRequestIDKey); requestID != nil {
		lc.z = lc.z.With(zap.Any(known.XRequestIDKey, requestID))
	}

	// 添加用户名
	if userID := ctx.Value(known.XUsernameKey); userID != nil {
		lc.z = lc.z.With(zap.Any(known.XUsernameKey, userID))
	}

	// 保持原始的 logger 实例
	lc.z = lc.z.With(zap.String("logger", "miniblog"))

	return lc
}

// clone 深拷贝 zapLogger 实例
func (l *zapLogger) clone() *zapLogger {
	lc := *l
	return &lc
}
