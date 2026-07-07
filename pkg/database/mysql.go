// Package database MySQL/GORM 连接池工厂。配置连接数、日志桥接、健康检查。
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"backend/pkg/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// NewMySQL 创建 GORM 连接池。步骤：日志级别 → GORM Config → 连接池参数 → Ping → 查版本
func NewMySQL(cfg config.DatabaseConfig, logLevel string, slowThreshold time.Duration) (*gorm.DB, error) {
	var gormLogLevel gormlogger.LogLevel
	switch logLevel {
	case "silent": gormLogLevel = gormlogger.Silent
	case "error":  gormLogLevel = gormlogger.Error
	case "warn":   gormLogLevel = gormlogger.Warn
	default:       gormLogLevel = gormlogger.Info
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(&gormLogWriter{}, gormlogger.Config{
			SlowThreshold:             slowThreshold,             // 慢查询阈值，超此值记 warn
			LogLevel:                  gormLogLevel,              // 日志级别
			IgnoreRecordNotFoundError: true,                      // 查不到不记 error
			Colorful:                  false,                     // 日志文件不要颜色
		}),
		SkipDefaultTransaction: true,  // 单行操作不自动包事务
		PrepareStmt:            true,  // 缓存预处理语句
		AllowGlobalUpdate:      false, // 禁止全表更新（安全护栏）
		DisableAutomaticPing:   false,
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil { return nil, fmt.Errorf("failed to connect database: %w", err) }

	sqlDB, err := db.DB()
	if err != nil { return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err) }

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	var version string
	if err := db.Raw("SELECT VERSION()").Scan(&version).Error; err != nil {
		slog.Warn("failed to get MySQL version", "error", err)
	} else {
		slog.Info("MySQL connected", "version", version)
	}
	return db, nil
}

// MustNewMySQL 失败时 panic，适用于 main 启动阶段
func MustNewMySQL(cfg config.DatabaseConfig, logLevel string, slowThreshold time.Duration) *gorm.DB {
	db, err := NewMySQL(cfg, logLevel, slowThreshold)
	if err != nil { panic(fmt.Sprintf("failed to init database: %v", err)) }
	return db
}

// Close 关闭数据库连接池（defer 中调用）
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil { return err }
	return sqlDB.Close()
}

// HealthCheck 2 秒超时 ping，预留供 HTTP 探活使用
func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil { return err }
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

// gormLogWriter 桥接 GORM Printf → slog。args[0] 为级别（error/warn/info）
type gormLogWriter struct{}

// Printf GORM 日志回调，按 args[0] 分类转发到 slog
func (w *gormLogWriter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if len(args) > 0 {
		switch args[0] {
		case "error": slog.Error(msg)
		case "warn":  slog.Warn(msg)
		default:      slog.Info(msg)
		}
	} else {
		slog.Info(msg)
	}
}
