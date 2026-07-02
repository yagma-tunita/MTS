package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
	JWT      JWTConfig      `yaml:"jwt"`
	Freight  FreightConfig  `yaml:"freight"`
	Notify   NotifyConfig   `yaml:"notify"`
}

type ServerConfig struct {
	Port         string        `yaml:"port" env:"SERVER_PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `yaml:"read_timeout" env:"SERVER_READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout time.Duration `yaml:"write_timeout" env:"SERVER_WRITE_TIMEOUT" envDefault:"30s"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" env:"SERVER_IDLE_TIMEOUT" envDefault:"60s"`
}

type DatabaseConfig struct {
	Driver          string        `yaml:"driver" env:"DB_DRIVER" envDefault:"mysql"`
	DSN             string        `yaml:"dsn" env:"DB_DSN" envDefault:"root:111322@qq.comxyZ@tcp(127.0.0.1:3306)/mts?charset=utf8mb4&parseTime=True&loc=Local"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"DB_MAX_OPEN_CONNS" envDefault:"25"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"DB_MAX_IDLE_CONNS" envDefault:"10"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"DB_CONN_MAX_LIFETIME" envDefault:"5m"`
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time" env:"DB_CONN_MAX_IDLE_TIME" envDefault:"5m"`
}

type LogConfig struct {
	Level      string `yaml:"level" env:"LOG_LEVEL" envDefault:"info"`
	Format     string `yaml:"format" env:"LOG_FORMAT" envDefault:"json"`
	OutputPath string `yaml:"output_path" env:"LOG_OUTPUT_PATH" envDefault:"stdout"`
	MaxSize    int    `yaml:"max_size" env:"LOG_MAX_SIZE" envDefault:"100"`
	MaxBackups int    `yaml:"max_backups" env:"LOG_MAX_BACKUPS" envDefault:"10"`
	MaxAge     int    `yaml:"max_age" env:"LOG_MAX_AGE" envDefault:"30"`
	Compress   bool   `yaml:"compress" env:"LOG_COMPRESS" envDefault:"true"`
}

type JWTConfig struct {
	Secret        string        `yaml:"secret" env:"JWT_SECRET" envDefault:"change-me-in-production"`
	AccessExpire  time.Duration `yaml:"access_expire" env:"JWT_ACCESS_EXPIRE" envDefault:"15m"`
	RefreshExpire time.Duration `yaml:"refresh_expire" env:"JWT_REFRESH_EXPIRE" envDefault:"168h"`
}

type FreightConfig struct {
	BaseRatePerTonNm float64            `yaml:"base_rate_per_ton_nm"`
	CargoTypeFactors map[string]float64 `yaml:"cargo_type_factors"`
}

type EmailNotifyConfig struct {
	SMTPHost string `yaml:"smtp_host" env:"NOTIFY_EMAIL_SMTP_HOST"`
	SMTPPort int    `yaml:"smtp_port" env:"NOTIFY_EMAIL_SMTP_PORT"`
	Username string `yaml:"username" env:"NOTIFY_EMAIL_USERNAME"`
	Password string `yaml:"password" env:"NOTIFY_EMAIL_PASSWORD"`
	FromAddr string `yaml:"from_addr" env:"NOTIFY_EMAIL_FROM_ADDR"`
	FromName string `yaml:"from_name" env:"NOTIFY_EMAIL_FROM_NAME"`
}

type SMSNotifyConfig struct {
	Provider        string `yaml:"provider" env:"NOTIFY_SMS_PROVIDER"`
	AccessKeyID     string `yaml:"access_key_id" env:"NOTIFY_SMS_ACCESS_KEY_ID"`
	AccessKeySecret string `yaml:"access_key_secret" env:"NOTIFY_SMS_ACCESS_KEY_SECRET"`
	SignName        string `yaml:"sign_name" env:"NOTIFY_SMS_SIGN_NAME"`
	TemplateCode    string `yaml:"template_code" env:"NOTIFY_SMS_TEMPLATE_CODE"`
}

type NotifyConfig struct {
	Email EmailNotifyConfig `yaml:"email"`
	SMS   SMSNotifyConfig   `yaml:"sms"`
}

var globalConfig *Config

func Load(configPath string) (*Config, error) {
	cfg := &Config{}
	applyDefaults(cfg)

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	overrideFromEnv(cfg)

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	globalConfig = cfg
	return cfg, nil
}

func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load first")
	}
	return globalConfig
}

func (c *Config) Validate() error {
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.Database.Driver == "" {
		return fmt.Errorf("database driver cannot be empty")
	}
	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN cannot be empty")
	}
	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}
	return nil
}

func applyDefaults(cfg *Config) {
	cfg.Server.Port = "8080"
	cfg.Server.ReadTimeout = 30 * time.Second
	cfg.Server.WriteTimeout = 30 * time.Second
	cfg.Server.IdleTimeout = 60 * time.Second

	cfg.Database.Driver = "mysql"
	cfg.Database.DSN = "root:111322@qq.comxyZ@tcp(127.0.0.1:3306)/mts?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.Database.MaxOpenConns = 25
	cfg.Database.MaxIdleConns = 10
	cfg.Database.ConnMaxLifetime = 5 * time.Minute
	cfg.Database.ConnMaxIdleTime = 5 * time.Minute

	cfg.Log.Level = "info"
	cfg.Log.Format = "json"
	cfg.Log.OutputPath = "stdout"
	cfg.Log.MaxSize = 100
	cfg.Log.MaxBackups = 10
	cfg.Log.MaxAge = 30
	cfg.Log.Compress = true

	cfg.JWT.Secret = "change-me-in-production"
	cfg.JWT.AccessExpire = 15 * time.Minute
	cfg.JWT.RefreshExpire = 168 * time.Hour

	cfg.Freight.BaseRatePerTonNm = 0.05
	cfg.Freight.CargoTypeFactors = map[string]float64{
		"bulk":      1.0,
		"container": 1.2,
		"liquid":    1.1,
	}
}

func overrideFromEnv(cfg *Config) {
	// Server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.Server.Port = v
	}
	if v := os.Getenv("SERVER_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("SERVER_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("SERVER_IDLE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.IdleTimeout = d
		}
	}

	// Database
	if v := os.Getenv("DB_DRIVER"); v != "" {
		cfg.Database.Driver = v
	}
	if v := os.Getenv("DB_DSN"); v != "" {
		cfg.Database.DSN = v
	}
	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxOpenConns = i
		}
	}
	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxIdleConns = i
		}
	}
	if v := os.Getenv("DB_CONN_MAX_LIFETIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.ConnMaxLifetime = d
		}
	}
	if v := os.Getenv("DB_CONN_MAX_IDLE_TIME"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Database.ConnMaxIdleTime = d
		}
	}

	// Log
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}
	if v := os.Getenv("LOG_OUTPUT_PATH"); v != "" {
		cfg.Log.OutputPath = v
	}
	if v := os.Getenv("LOG_MAX_SIZE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Log.MaxSize = i
		}
	}
	if v := os.Getenv("LOG_MAX_BACKUPS"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Log.MaxBackups = i
		}
	}
	if v := os.Getenv("LOG_MAX_AGE"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Log.MaxAge = i
		}
	}
	if v := os.Getenv("LOG_COMPRESS"); v != "" {
		cfg.Log.Compress = strings.ToLower(v) == "true" || v == "1"
	}

	// JWT
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("JWT_ACCESS_EXPIRE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.AccessExpire = d
		}
	}
	if v := os.Getenv("JWT_REFRESH_EXPIRE"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.JWT.RefreshExpire = d
		}
	}

	// Freight
	if v := os.Getenv("FREIGHT_BASE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Freight.BaseRatePerTonNm = f
		}
	}
	if v := os.Getenv("FREIGHT_CARGO_FACTORS"); v != "" {
		var factors map[string]float64
		if err := json.Unmarshal([]byte(v), &factors); err == nil {
			cfg.Freight.CargoTypeFactors = factors
		}
	}

	// Notify
	if v := os.Getenv("NOTIFY_EMAIL_SMTP_HOST"); v != "" {
		cfg.Notify.Email.SMTPHost = v
	}
	if v := os.Getenv("NOTIFY_EMAIL_SMTP_PORT"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			cfg.Notify.Email.SMTPPort = i
		}
	}
	if v := os.Getenv("NOTIFY_EMAIL_USERNAME"); v != "" {
		cfg.Notify.Email.Username = v
	}
	if v := os.Getenv("NOTIFY_EMAIL_PASSWORD"); v != "" {
		cfg.Notify.Email.Password = v
	}
	if v := os.Getenv("NOTIFY_EMAIL_FROM_ADDR"); v != "" {
		cfg.Notify.Email.FromAddr = v
	}
	if v := os.Getenv("NOTIFY_EMAIL_FROM_NAME"); v != "" {
		cfg.Notify.Email.FromName = v
	}
	if v := os.Getenv("NOTIFY_SMS_PROVIDER"); v != "" {
		cfg.Notify.SMS.Provider = v
	}
}
