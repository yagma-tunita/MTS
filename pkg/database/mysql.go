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

func NewMySQL(cfg config.DatabaseConfig, logLevel string, slowThreshold time.Duration) (*gorm.DB, error) {
	var gormLogLevel gormlogger.LogLevel
	switch logLevel {
	case "silent":
		gormLogLevel = gormlogger.Silent
	case "error":
		gormLogLevel = gormlogger.Error
	case "warn":
		gormLogLevel = gormlogger.Warn
	default:
		gormLogLevel = gormlogger.Info
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.New(
			&gormLogWriter{},
			gormlogger.Config{
				SlowThreshold:             slowThreshold,
				LogLevel:                  gormLogLevel,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		AllowGlobalUpdate:      false,
		DisableAutomaticPing:   false,
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

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

func MustNewMySQL(cfg config.DatabaseConfig, logLevel string, slowThreshold time.Duration) *gorm.DB {
	db, err := NewMySQL(cfg, logLevel, slowThreshold)
	if err != nil {
		panic(fmt.Sprintf("failed to init database: %v", err))
	}
	return db
}

func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func HealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return sqlDB.PingContext(ctx)
}

type gormLogWriter struct{}

func (w *gormLogWriter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if len(args) > 0 {
		switch args[0] {
		case "error":
			slog.Error(msg)
		case "warn":
			slog.Warn(msg)
		default:
			slog.Info(msg)
		}
	} else {
		slog.Info(msg)
	}
}
