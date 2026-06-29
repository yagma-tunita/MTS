================================================================================
                PKG 公共基础设施模块总览
                版本：1.0
                生成日期：2026-06-09
================================================================================

本目录（backend/pkg）存放所有与业务无关的、可复用的基础设施代码。
各子模块职责单一，无循环依赖，不依赖 internal 目录，可被项目任意包导入。

--------------------------------------------------------------------------------
1. 模块列表及功能说明
--------------------------------------------------------------------------------

┌─────────────┬────────────────────────────────────────────────────────────────┐
│ 子模块       │ 功能                                                           │
├─────────────┼────────────────────────────────────────────────────────────────┤
│ config      │ 配置管理：从 YAML 文件和环境变量加载配置，支持默认值、验证。      │
│ logger      │ 结构化日志：基于 slog + lumberjack，支持级别控制、文件轮转、压缩 │
│ database    │ MySQL 连接池：GORM 初始化、健康检查、慢查询日志、连接参数调优     │
│ response    │ 统一 HTTP 响应：标准 JSON 格式（code/message/data/meta）        │
│ validator   │ 参数校验：基于 go-playground/validator，内置日期、手机号等规则   │
│ crypto      │ 密码学工具：bcrypt 哈希、AES-GCM 加密、MD5、安全随机字符串       │
│ jwt         │ JWT 认证：生成/解析/刷新令牌（HMAC-SHA256）                      │
│ errors      │ 业务错误：带错误码和堆栈的结构化错误，预定义 HTTP 风格构造函数   │
│ fileutil    │ 文件操作：原子写、目录拷贝、文件锁（跨平台）                      │
│ timeutil    │ 时间工具：东八区时间、日期解析/格式化、年龄计算                   │
│ idgen       │ 唯一 ID：雪花算法生成分布式唯一 ID（uint64）                     │
└─────────────┴────────────────────────────────────────────────────────────────┘

--------------------------------------------------------------------------------
2. 各模块使用示例
--------------------------------------------------------------------------------

2.1 config
-------------------------------------------------------------------------------
    import "backend/pkg/config"

    cfg, err := config.Load("config.yaml")
    // 或 config.MustLoad("config.yaml") 失败时 panic
    fmt.Println(cfg.Server.Port)

    配置文件示例（config.yaml）：
        server:
          port: "8080"
        database:
          dsn: "root:pass@tcp(localhost:3306)/db?parseTime=true"

    环境变量覆盖：DB_DSN="..." 会覆盖文件中的值。

2.2 logger
-------------------------------------------------------------------------------
    import "backend/pkg/logger"

    logger.Init("info", "json", "logs/app.log", 100, 10, 30, true)
    logger.Info("server started", "port", 8080)
    logger.With("user_id", 123).Debug("processing request")

2.3 database
-------------------------------------------------------------------------------
    import "backend/pkg/database"

    db, err := database.NewMySQL(cfg.Database, "info", 200*time.Millisecond)
    // 或 database.MustNewMySQL(...)
    sqlDB, _ := db.DB()
    sqlDB.SetMaxOpenConns(25)

2.4 response
-------------------------------------------------------------------------------
    import "backend/pkg/response"

    // 成功
    response.Success(w, data)

    // 分页成功
    response.SuccessPage(w, list, page, pageSize, total)

    // 错误
    response.BadRequest(w, "invalid param")
    response.NotFound(w, "resource not found")

2.5 validator
-------------------------------------------------------------------------------
    import "backend/pkg/validator"

    type CreateReq struct {
        Name     string `json:"name" validate:"required,min=2"`
        Email    string `json:"email" validate:"required,email"`
        Birth    string `json:"birth" validate:"date"`
    }
    var req CreateReq
    if err := validator.Validate(req); err != nil {
        // 处理错误（可获取具体字段）
    }

2.6 crypto
-------------------------------------------------------------------------------
    import "backend/pkg/crypto"

    // 密码哈希
    hash, _ := crypto.HashPassword("myPass")
    ok := crypto.CheckPasswordHash("myPass", hash)

    // AES 加密
    key := make([]byte, 32)
    ciphertext, _ := crypto.AESEncrypt(key, []byte("secret"))
    plain, _ := crypto.AESDecrypt(key, ciphertext)

    // 随机字符串
    token, _ := crypto.GenerateRandomString(32)

2.7 jwt
-------------------------------------------------------------------------------
    import "backend/pkg/jwt"

    jwtSvc := jwt.NewJWTService("secret-key", 15*time.Minute, 7*24*time.Hour)
    accessToken, _ := jwtSvc.GenerateAccessToken(1, "admin", "admin")
    claims, err := jwtSvc.ValidateToken(accessToken)

2.8 errors
-------------------------------------------------------------------------------
    import pkgerr "backend/pkg/errors"

    // 预定义错误
    err := pkgerr.NotFound("user not found")
    err := pkgerr.BadRequest("invalid param")
    err := pkgerr.Internal("database error")
    err := pkgerr.Wrap(pkgerr.CodeDatabaseError, "query failed", originalErr)

    // 获取错误码
    if appErr, ok := err.(*pkgerr.AppError); ok {
        fmt.Println(appErr.Code, appErr.Message)
    }

2.9 fileutil
-------------------------------------------------------------------------------
    import "backend/pkg/fileutil"

    fileutil.EnsureDir("./data", 0755)
    fileutil.WriteFile("./data/config.json", []byte("{}"), 0644)
    lock := fileutil.NewFileLock("/tmp/mylock.lock")
    lock.Lock()
    defer lock.Unlock()

2.10 timeutil
-------------------------------------------------------------------------------
    import "backend/pkg/timeutil"

    now := timeutil.Now()   // 东八区当前时间
    t, _ := timeutil.ParseDate("2026-06-09")
    formatted := timeutil.FormatDateTime(t)
    age := timeutil.Age(birthDate)

2.11 idgen
-------------------------------------------------------------------------------
    import "backend/pkg/idgen"

    id := idgen.NextID()   // uint64，雪花算法
    fmt.Println(id)

--------------------------------------------------------------------------------
3. 模块依赖关系
--------------------------------------------------------------------------------

所有 pkg 子模块仅依赖标准库和少量精选第三方库，无内部循环依赖。

第三方依赖列表：
    - gopkg.in/yaml.v3          (config)
    - gopkg.in/natefinch/lumberjack.v2 (logger)
    - github.com/go-playground/validator/v10 (validator)
    - github.com/golang-jwt/jwt/v5 (jwt)
    - github.com/sony/sonyflake (idgen)
    - github.com/gofrs/flock     (fileutil)
    - gorm.io/gorm               (database)
    - gorm.io/driver/mysql       (database)

依赖关系图：
    config  ──┬──> (无)
    logger   ──┬──> lumberjack
    database ──┬──> gorm + mysql driver
    response ──┬──> (无)
    validator ──┬──> go-playground/validator
    crypto   ──┬──> bcrypt (golang.org/x/crypto)
    jwt      ──┬──> golang-jwt
    errors   ──┬──> (无)
    fileutil ──┬──> flock
    timeutil ──┬──> (无)
    idgen    ──┬──> sonyflake

各模块可以被任何其他包导入，不会产生循环引用。

--------------------------------------------------------------------------------
4. 初始化顺序建议（在 main.go 中）
--------------------------------------------------------------------------------

    1. config.Load()        // 必须最先，为其他模块提供配置
    2. logger.Init()        // 日志初始化，之后可以使用日志
    3. database.NewMySQL()  // 数据库连接（依赖 config）
    4. 其他初始化（JWT 服务、ID 生成器等）

示例 main.go 片段：
    cfg := config.MustLoad("config.yaml")
    logger.Init(cfg.Log.Level, cfg.Log.Format, cfg.Log.OutputPath,
        cfg.Log.MaxSize, cfg.Log.MaxBackups, cfg.Log.MaxAge, cfg.Log.Compress)
    db := database.MustNewMySQL(cfg.Database, cfg.Log.Level, 200*time.Millisecond)
    jwtSvc := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)

--------------------------------------------------------------------------------
5. 错误处理约定
--------------------------------------------------------------------------------

- 所有可能失败的操作都应返回 error，优先使用 pkg/errors 中的 AppError 类型。
- 下层模块（如 config、database）返回标准 error，但包装了足够上下文。
- HTTP 层使用 response 包构造统一响应，业务错误码来自 pkg/errors。
- 日志记录统一通过 pkg/logger 或 slog.Default() 完成。

--------------------------------------------------------------------------------
6. 配置说明（环境变量优先）
--------------------------------------------------------------------------------

常用环境变量：
    SERVER_PORT          HTTP 监听端口，默认 8080
    DB_DSN               MySQL 数据源名称
    LOG_LEVEL            debug/info/warn/error，默认 info
    LOG_OUTPUT_PATH      日志输出路径，stdout 或文件路径，默认 stdout
    JWT_SECRET           JWT 签名密钥（生产环境必须修改）

所有配置项均支持通过 YAML 文件设置，环境变量会覆盖文件值。

--------------------------------------------------------------------------------
7. 注意事项
--------------------------------------------------------------------------------

- 所有 pkg 模块均无中文注释，仅使用英文。
- 禁止在 pkg 中引入 internal 中的任何包。
- 新增公共工具应优先考虑放入 pkg 对应子模块，保持职责单一。
- 单元测试覆盖率应达到 80% 以上（当前提供核心功能，测试需自行补充）。
- 生产环境必须修改 JWT_SECRET 默认值，避免安全风险。

2026-06-11 更新内容 (pkg 模块)

1. 新增 cache 模块
   - 位置: pkg/cache/
   - 功能: 内存缓存（基于 go-cache），支持 TTL、自动清理、前缀删除。
   - 使用示例:
       import "backend/pkg/cache"
       cache.Set("key", value, 5*time.Minute)
       if val, found := cache.Get("key"); found { ... }
       cache.DeletePrefix("voyage_rec:")

2. 新增 excel 模块
   - 位置: pkg/excel/
   - 功能: Excel 读写工具（基于 excelize），支持读取 sheet、写入数据、自动调整列宽。
   - 使用示例:
       import "backend/pkg/excel"
       rows, err := excel.ReadSheet(file, fileSize)
       data, err := excel.WriteSheet(headers, rows)
================================================================================
                              文档结束
================================================================================