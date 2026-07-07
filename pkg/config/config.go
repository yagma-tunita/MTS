// Package config 提供应用程序配置管理功能。
//
// 配置来源优先级（低 → 高）：
//   1. 默认值（由 applyDefaults 函数硬编码）
//   2. YAML 配置文件（通过 yaml.Unmarshal 解析，覆盖默认值）
//   3. 环境变量（通过 overrideFromEnv 读取 os.Getenv，覆盖文件配置）
//
// 设计理念：
//   - 所有配置聚合在单一的 Config 结构体中，通过 Get() 函数全局访问。
//   - 环境变量命名规则为 {SECTION}_{KEY}，例如 DB_DSN、JWT_SECRET。
//   - 使用 Load → MustLoad 模式：失败时 panic 或返回 error 任由调用者选择。
//   - Validate() 确保必填项（数据库 DSN、JWT 密钥等）在启动时即被检查，避免运行时才暴露问题。
//
// 推荐初始化顺序（在 main.go 中）：
//   cfg := config.MustLoad("config.yaml")
//   // 之后读取 cfg.Server.Port、cfg.Database.DSN 等
//
// 生产环境注意：
//   - JWT_SECRET 必须修改，不可使用默认值 "change-me-in-production"。
//   - 敏感信息（密码、密钥）优先通过环境变量注入，而非写入 YAML 文件。
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

// ──────────────────────────────────────────────────────────────────────────────
// 配置结构体定义
// ──────────────────────────────────────────────────────────────────────────────

// Config 是应用顶层配置结构体，聚合所有功能模块的配置子项。
// 每个子结构体对应一个配置领域，与 YAML 文件中的顶级键一一对应。
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // HTTP 服务器配置
	Database DatabaseConfig `yaml:"database"` // MySQL 数据库连接配置
	Log      LogConfig      `yaml:"log"`      // 结构化日志配置
	JWT      JWTConfig      `yaml:"jwt"`      // JWT 认证配置
	Freight  FreightConfig  `yaml:"freight"`  // 运费计算费率配置
	Notify   NotifyConfig   `yaml:"notify"`   // 通知服务配置（邮件 + SMS）
}

// ServerConfig HTTP 服务器相关配置。
// 对应 config.yaml 中的 server 段。
type ServerConfig struct {
	Port         string        `yaml:"port"`          // 监听端口，如 "8080"。默认值在 applyDefaults 中设置。
	ReadTimeout  time.Duration `yaml:"read_timeout"`  // 读取请求的最大时长，超过则断开连接。默认 30s。
	WriteTimeout time.Duration `yaml:"write_timeout"` // 写入响应的最大时长，超过则断开连接。默认 30s。
	IdleTimeout  time.Duration `yaml:"idle_timeout"`  // Keep-Alive 空闲连接的最大等待时长。默认 60s。
}

// DatabaseConfig MySQL 数据库连接池配置。
// 对应 config.yaml 中的 database 段。
// DSN 格式参考：username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
type DatabaseConfig struct {
	Driver          string        `yaml:"driver"`            // 数据库驱动，固定为 "mysql"。保留扩展性。
	DSN             string        `yaml:"dsn"`               // 数据源名称，含用户名密码和连接参数。
	MaxOpenConns    int           `yaml:"max_open_conns"`    // 连接池最大打开连接数。默认 25。过高可能导致 MySQL 连接数耗尽。
	MaxIdleConns    int           `yaml:"max_idle_conns"`    // 连接池最大空闲连接数。默认 10。建议不超过 MaxOpenConns。
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"` // 连接最长存活时间，超过后会被主动关闭。默认 5 分钟。防止 MySQL 8 小时超时断开。
	ConnMaxIdleTime time.Duration `yaml:"conn_max_idle_time"` // 连接最长空闲时间，超过后回收。默认 5 分钟。
}

// LogConfig 结构化日志输出配置。
// 对应 config.yaml 中的 log 段。
// 使用 Go 1.21 标准库 log/slog 实现，支持 JSON 和 Text 两种格式。
// 文件轮转基于 lumberjack 库：按大小分割、保留指定数量的备份、可选压缩。
type LogConfig struct {
	Level      string `yaml:"level"`       // 日志最低级别：debug/info/warn/error。默认 info。
	Format     string `yaml:"format"`      // 输出格式：json/text。默认 json（方便日志收集系统解析）。
	OutputPath string `yaml:"output_path"` // 日志文件路径。设为 "stdout" 或空则只输出到控制台。
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大大小（MB），超过则轮转。默认 100MB。
	MaxBackups int    `yaml:"max_backups"` // 保留的旧日志文件数量上限。默认 10 个。
	MaxAge     int    `yaml:"max_age"`     // 旧日志文件保留天数。默认 30 天。
	Compress   bool   `yaml:"compress"`    // 是否压缩旧日志文件（gzip）。默认 true。
}

// JWTConfig JSON Web Token 认证配置。
// 对应 config.yaml 中的 jwt 段。
type JWTConfig struct {
	Secret        string        `yaml:"secret"`         // HMAC-SHA256 签名密钥。生产环境必须修改默认值！
	AccessExpire  time.Duration `yaml:"access_expire"`  // Access Token 过期时间。默认 15 分钟。短时效降低泄露风险。
	RefreshExpire time.Duration `yaml:"refresh_expire"` // Refresh Token 过期时间。默认 168 小时（7 天）。
}

// FreightConfig 运费费率配置，用于计算运输费用。
// 对应 config.yaml 中的 freight 段。
// 运费计算公式：总费用 = 总重量(吨) × 总航程(海里) × 基础费率 × 货物类型系数
type FreightConfig struct {
	BaseRatePerTonNm float64            `yaml:"base_rate_per_ton_nm"` // 每吨每海里的基础费率（元）。
	CargoTypeFactors map[string]float64 `yaml:"cargo_type_factors"`   // 货物类型调整系数。
	// 默认值：bulk=1.0（散货）, container=1.2（集装箱，因操作复杂度高）, liquid=1.1（液体）。
	// 实际费率 = BaseRatePerTonNm * CargoTypeFactor。
}

// EmailNotifyConfig 邮件通知的 SMTP 服务器配置。
// 支持 STARTTLS（端口 587）和 SSL/TLS（端口 465）两种方式。
type EmailNotifyConfig struct {
	SMTPHost string `yaml:"smtp_host"` // SMTP 服务器地址，如 smtp.gmail.com。
	SMTPPort int    `yaml:"smtp_port"` // SMTP 端口。587=STARTTLS，465=SSL。
	Username string `yaml:"username"`  // SMTP 登录用户名（通常与发件人邮箱相同）。
	Password string `yaml:"password"`  // SMTP 登录密码或应用专用密码。
	FromAddr string `yaml:"from_addr"` // 发件人邮箱地址。
	FromName string `yaml:"from_name"` // 发件人显示名称。
}

// SMSNotifyConfig 短信通知的云服务商配置。
// 当前支持阿里云和腾讯云两家国内主流短信服务商。
type SMSNotifyConfig struct {
	Provider        string `yaml:"provider"`          // 短信服务商：aliyun / tencent / console（仅打印日志）。
	AccessKeyID     string `yaml:"access_key_id"`     // API 访问密钥 ID。
	AccessKeySecret string `yaml:"access_key_secret"` // API 访问密钥 Secret。
	SignName        string `yaml:"sign_name"`         // 短信签名（需在服务商后台审核通过）。
	TemplateCode    string `yaml:"template_code"`     // 短信模板编码。
}

// NotifyConfig 聚合邮件和短信通知的完整配置。
type NotifyConfig struct {
	Email EmailNotifyConfig `yaml:"email"`
	SMS   SMSNotifyConfig   `yaml:"sms"`
}

// ──────────────────────────────────────────────────────────────────────────────
// 配置加载与访问
// ──────────────────────────────────────────────────────────────────────────────

// globalConfig 全局配置实例，通过 Load 或 MustLoad 设置。
// 为什么使用包级变量而不是传参：业务层（如订单服务需要读运费费率配置）遍布各处，
// 每次传递 config 对象会增加签名复杂度，通过 Get() 单例访问最简便。
// 缺点是必须在启动时先调用 Load，Get() 在 Load 之前调用会 panic。
var globalConfig *Config

// Load 从 YAML 文件加载配置，并用环境变量覆盖。
//
// 处理流程（按顺序执行，后面的覆盖前面的）：
//  1. applyDefaults(cfg) —— 设置所有字段的硬编码默认值。
//  2. 读取 YAML 文件 —— 如果 configPath 不为空且文件存在，用 yaml.Unmarshal 解析并覆盖默认值。
//  3. overrideFromEnv(cfg) —— 遍历所有支持的环境变量（与 struct 的 env tag 对应），
//     非空则覆盖已加载的值（包括 YAML 中的值）。
//  4. Validate() —— 检查必填项，缺少则返回错误。
//  5. 赋值 globalConfig —— 使 Get() 可访问。
//
// 参数 configPath 是 YAML 文件的路径，可以为空（仅使用默认值+环境变量）。
//
// 返回值：
//   - *Config: 加载成功的配置指针。
//   - error: 文件读取、解析或校验失败时返回错误。
func Load(configPath string) (*Config, error) {
	cfg := &Config{}
	applyDefaults(cfg)

	// 读取 YAML 文件（如果存在）。文件不存在不报错，只使用默认值+环境变量。
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

// MustLoad 是 Load 的便捷版本，配置加载失败时直接 panic（终止程序）。
// 适用于 main 函数启动阶段——如果连配置都加载不了，后续逻辑也无法运行。
func MustLoad(configPath string) *Config {
	cfg, err := Load(configPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}
	return cfg
}

// Get 返回全局配置实例。必须在 Load 或 MustLoad 之后调用，否则 panic。
// 为什么设计成可 panic：配置是系统的基础依赖，如果配置未加载就尝试使用，
// 一定是编码错误，应该尽早暴露而不是静默返回 nil 导致后续空指针异常。
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded, call Load first")
	}
	return globalConfig
}

// ──────────────────────────────────────────────────────────────────────────────
// 校验
// ──────────────────────────────────────────────────────────────────────────────

// Validate 检查必填配置项是否已设置。
// 四个必填项：Server.Port（监听端口）、Database.Driver（驱动）、
// Database.DSN（数据库连接串）、JWT.Secret（签名密钥）。
// 注意：本函数只做非空检查。各子模块（如 database.NewMySQL）会在使用时
// 检查配置值的有效性（如端口格式、DSN 语法等）。
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

// ──────────────────────────────────────────────────────────────────────────────
// 默认值
// ──────────────────────────────────────────────────────────────────────────────

// applyDefaults 设置所有配置项的默认值。
// 这些默认值是最低保障，后续会被 YAML 文件和环境变量逐层覆盖。
// 为什么需要默认值：让开发者在没有配置文件的情况下也能启动应用（开发调试）。
// 为什么不在 struct tag 中用 default 标注：Go 标准库不支持这种能力。
func applyDefaults(cfg *Config) {
	// Server 默认值：端口 8080，超时时间与常见反向代理（如 Nginx）的超时设置匹配。
	cfg.Server.Port = "8080"
	cfg.Server.ReadTimeout = 30 * time.Second
	cfg.Server.WriteTimeout = 30 * time.Second
	cfg.Server.IdleTimeout = 60 * time.Second

	// Database 默认值：本地开发用的 MySQL root 账号，注意用户名密码为占位符。
	cfg.Database.Driver = "mysql"
	cfg.Database.DSN = "root:your_password@tcp(127.0.0.1:3306)/mts?charset=utf8mb4&parseTime=True&loc=Local"
	cfg.Database.MaxOpenConns = 25
	cfg.Database.MaxIdleConns = 10
	cfg.Database.ConnMaxLifetime = 5 * time.Minute
	cfg.Database.ConnMaxIdleTime = 5 * time.Minute

	// Log 默认值：JSON 格式（方便日志系统如 ELK/Loki 解析），输出到 stdout（Docker 友好）。
	cfg.Log.Level = "info"
	cfg.Log.Format = "json"
	cfg.Log.OutputPath = "stdout"
	cfg.Log.MaxSize = 100
	cfg.Log.MaxBackups = 10
	cfg.Log.MaxAge = 30
	cfg.Log.Compress = true

	// JWT 默认值：密钥为占位符，生产环境必须修改；access token 15 分钟，refresh token 7 天。
	cfg.JWT.Secret = "change-me-in-production"
	cfg.JWT.AccessExpire = 15 * time.Minute
	cfg.JWT.RefreshExpire = 168 * time.Hour

	// Freight 默认值：基础费率 0.05 元/吨海里，三种货物类型的调整系数。
	cfg.Freight.BaseRatePerTonNm = 0.05
	cfg.Freight.CargoTypeFactors = map[string]float64{
		"bulk":      1.0,  // 散货：基准值
		"container": 1.2,  // 集装箱：因需要专用设备和额外操作，费率上浮 20%
		"liquid":    1.1,  // 液体：费率上浮 10%
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// 环境变量覆盖
// ──────────────────────────────────────────────────────────────────────────────

// overrideFromEnv 读取环境变量并覆盖配置中的对应字段。
//
// 为什么手动读取而不是使用第三方库（如 viper）：
//   - 减少外部依赖，代码清晰可控。
//   - 所有支持的 env vars 一目了然。
//   - 可以精确控制类型转换（如 ParseDuration、strconv.Atoi）和解析失败的静默处理。
//
// 类型转换规则：
//   - string: 直接赋值。
//   - int: 使用 strconv.Atoi，失败则跳过（不报错，保留原值）。
//   - float64: 使用 strconv.ParseFloat，失败则跳过。
//   - time.Duration: 使用 time.ParseDuration，失败则跳过。
//   - bool: "true"/"1" 视为 true，其余视为 false。
//   - map: 使用 json.Unmarshal 解析 JSON 字符串。
func overrideFromEnv(cfg *Config) {
	// ── 服务器配置 ──
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

	// ── 数据库配置 ──
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

	// ── 日志配置 ──
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

	// ── JWT 配置 ──
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

	// ── 运费配置 ──
	if v := os.Getenv("FREIGHT_BASE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Freight.BaseRatePerTonNm = f
		}
	}
	if v := os.Getenv("FREIGHT_CARGO_FACTORS"); v != "" {
		// 环境变量是 JSON 字符串，如 '{"bulk":1.0,"container":1.5}'
		var factors map[string]float64
		if err := json.Unmarshal([]byte(v), &factors); err == nil {
			cfg.Freight.CargoTypeFactors = factors
		}
	}

	// ── 通知配置 ──
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
	// SMS 其余字段（AccessKeyID、AccessKeySecret 等）的 env 覆盖未实现，
	// 如需要可参照上述模式添加。当前仅 Provider 支持环境变量。
}
