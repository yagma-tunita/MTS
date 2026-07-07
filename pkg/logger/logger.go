// Package logger 提供结构化日志初始化和便捷函数。
//
// 基于 Go 1.21 标准库 log/slog，不再使用第三方日志库（如 logrus、zap）。
//
// 为什么选 slog 而不是 zap/logrus：
//   - Go 1.21 标准库内置，零依赖。
//   - 支持结构化输出（JSON/Text）和级别过滤，满足当前业务需求。
//   - 当前系统的日志量（每秒几千条）远未达到需要 zap 极致性能的量级。
//   - 少一个第三方依赖就少一个潜在的安全漏洞和维护负担。
//
// 输出方式：
//   - 始终同时输出到 stdout（容器友好，Docker/K8s 直接捕获 stdout）。
//   - 可选同时输出到文件，基于 lumberjack 实现轮转：
//     按大小分割（默认 100MB）、保留指定备份数（10 个）、
//     按时间清理（30 天）、可启用 gzip 压缩。
//
// 使用示例：
//
//	logger.Init("info", "json", "logs/app.log", 100, 10, 30, true)
//	logger.Info("server started", "port", 8080)
//	logger.With("request_id", "abc-123").Debug("processing request")
//
// Init 应该在 main 函数开头调用，之后全局使用 slog 或 logger 包的函数。
package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Init 初始化全局日志器。
//
// 参数说明：
//   - level: 日志最低级别。低于此级别的日志不会被输出。
//     可选值：debug / info / warn / error。不匹配时默认 info。
//     建议开发环境用 debug，生产环境用 info。
//   - format: 输出格式。可选值：
//     "json" —— JSON 格式，每行一个 JSON 对象，适合日志收集系统（ELK/Loki）。
//     "text" —— 纯文本格式，可读性好，适合开发调试。
//   - outputPath: 日志文件路径。特殊值：
//     "" 或 "stdout" —— 只输出到控制台，不写文件。
//     其他值（如 "logs/app.log"）—— 同时输出到控制台和文件。
//   - maxSize: 单个日志文件的最大大小（MB）。超过此值会自动轮转。
//     默认值 100 MB。如果日志量很大，可以调小以更快轮转。
//   - maxBackups: 保留的旧日志文件最大数量。超过此数的旧文件会被删除。
//     设为 0 表示保留所有旧文件（可能占用大量磁盘空间）。
//   - maxAge: 旧日志文件保留天数。超过此天数的自动删除。
//     轮转后的文件不会立即删除，而是等到清理周期（每天一次）到达。
//   - compress: 是否对轮转后的旧文件进行 gzip 压缩。
//     启用可以减少磁盘占用，但读取旧日志时需要先解压。
//
// 初始化流程：
//  1. 解析字符串日志级别为 slog.Level 枚举。
//  2. 构建 io.MultiWriter：始终包含 os.Stdout，
//     如果 outputPath 有效则添加 lumberjack.Logger。
//  3. 根据 format 创建 JSON 或 Text handler。
//  4. 调用 slog.SetDefault 将创建的 logger 设为包级默认。
//
// 注意：Init 不是线程安全的，应在 main 函数的最开头单线程调用。
// 多次调用 Init 会覆盖之前设置的默认 logger。
func Init(level string, format string, outputPath string, maxSize, maxBackups, maxAge int, compress bool) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// 构建输出目标：始终包含 stdout，可选添加文件（lumberjack）
	var writers []io.Writer
	writers = append(writers, os.Stdout)
	if outputPath != "" && outputPath != "stdout" {
		writers = append(writers, &lumberjack.Logger{
			Filename:   outputPath,
			MaxSize:    maxSize,
			MaxBackups: maxBackups,
			MaxAge:     maxAge,
			Compress:   compress,
		})
	}

	writer := io.MultiWriter(writers...)

	opts := &slog.HandlerOptions{Level: logLevel}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(writer, opts)
	} else {
		handler = slog.NewTextHandler(writer, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// Info 输出 info 级别日志。参数 args 为交替的键值对。
//
// 示例：
//
//	logger.Info("user logged in", "user_id", 123, "role", "admin")
//	// 输出: {"time":"...","level":"INFO","msg":"user logged in","user_id":123,"role":"admin"}
//
// args 必须是偶数个，奇数个会导致最后一个 key 没有 value 被忽略。
func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

// Debug 输出 debug 级别日志。仅在日志级别设为 debug 时才会输出。
// 用于开发调试信息，生产环境通常关闭。
func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}

// Warn 输出 warn 级别日志。用于需要关注的异常情况，但不至于影响服务。
// 如：慢查询、非致命的外部调用失败等。
func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

// Error 输出 error 级别日志。用于需要立即关注的错误。
// 如：数据库连接失败、关键操作异常等。
func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

// With 返回一个附加了指定上下文属性的 Logger。
// 返回的 Logger 会记录所有预定义的属性，避免在每个日志调用中重复传入。
//
// 示例：
//
//	reqLogger := logger.With("request_id", "abc-123", "user_id", 456)
//	reqLogger.Info("processing request") // 自动携带 request_id 和 user_id
//	reqLogger.Debug("db query", "sql", "SELECT ...") // 也携带 request_id 和 user_id
//
// 注意：With 返回新的 Logger 实例，不会修改原来的默认 Logger。
func With(args ...any) *slog.Logger {
	return slog.Default().With(args...)
}
