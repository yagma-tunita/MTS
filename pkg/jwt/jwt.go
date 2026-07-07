// Package jwt 提供 JWT（JSON Web Token）认证服务。
//
// 认证机制：
//   - 双 Token 模式：access_token（短期，15 分钟） + refresh_token（长期，7 天）。
//   - 签名算法：HMAC-SHA256（HS256），对称密钥。
//   - Token 中携带用户 ID、用户名、角色，用于后续请求的认证和鉴权。
//
// 为什么用双 Token 而不是单 Token：
//   - 单 Token 如果过期时间短（15 分钟），用户体验差（频繁重新登录）。
//   - 单 Token 如果过期时间长（7 天），泄露风险大（黑客有 7 天时间窗口）。
//   - 双 Token 折中：access_token 短命，即使泄露也只影响 15 分钟；
//     refresh_token 长命但只用于换 access_token，不能直接操作用户数据。
//
// 为什么不使用 RSA/ECDSA 非对称签名：
//   - 非对称签名可以用公钥验证、私钥签名，适合微服务间认证。
//   - 当前系统是单体应用，没有其他服务需要验证 token，
//     使用 HMAC-SHA256 对称签名更简单高效。
//   - 如果将来拆分成微服务，可以升级为 RS256 或 ES256。
//
// Token 数据结构（CustomClaims）：
//   - user_id:  用户 ID（货主公司/船公司/管理员）。
//   - username: 登录用户名。
//   - role:     角色— shipper(货主) / shipping(船公司) / admin(管理员)。
//   - 标准字段：ExpiresAt(过期时间)、IssuedAt(签发时间)、NotBefore(生效时间)。
//
// 使用示例：
//
//	jwtSvc := jwt.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessExpire, cfg.JWT.RefreshExpire)
//	accessToken, _ := jwtSvc.GenerateAccessToken(1, "test001", "shipper")
//	claims, err := jwtSvc.ValidateToken(accessToken)
//	fmt.Println(claims.Role) // "shipper"
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 定义 JWT 载荷（Payload）中的自定义字段。
//
// 嵌入了 jwt.RegisteredClaims（官方标准字段）：
//   - ExpiresAt: 过期时间（NumericDate）
//   - IssuedAt:  签发时间
//   - NotBefore: 生效时间（在此之前 token 不可用）
//   - Issuer:    签发者（当前未使用）
//   - Subject:   主题（当前未使用）
//   - ID:        令牌 ID（当前未使用）
//
// 自定义字段使用 json tag 做小写命名，符合 JWT 命名惯例。
type CustomClaims struct {
	UserID   int64  `json:"user_id"`  // 用户 ID
	Username string `json:"username"` // 登录用户名
	Role     string `json:"role"`     // 角色：shipper(货主) / shipping(船公司) / admin(管理员)
	jwt.RegisteredClaims
}

// JWTService 定义 JWT 操作的完整接口。
//
// 四个方法对应 JWT 的完整生命周期：
//   1. 签发 Access Token（短期）
//   2. 签发 Refresh Token（长期）
//   3. 验证 Token（解析并校验签名和有效期）
//   4. 刷新 Access Token（用 Refresh Token 换取新的 Access Token）
type JWTService interface {
	GenerateAccessToken(userID int64, username, role string) (string, error)
	GenerateRefreshToken(userID int64, username, role string) (string, error)
	ValidateToken(tokenString string) (*CustomClaims, error)
	RefreshAccessToken(refreshTokenString string) (string, error)
}

// jwtService 是 JWTService 接口的私有实现。
//
// 为什么是私有 struct 而不是导出：所有外部代码应该通过接口
// JWTService 使用，而非直接依赖具体实现。这样将来可以替换
// 为其他 JWT 库而不影响调用方。
type jwtService struct {
	secret        []byte        // HMAC-SHA256 签名密钥（[]byte 类型便于直接传入 jwt.SignedString）
	accessExpire  time.Duration // Access Token 过期时间，从签发时开始计算
	refreshExpire time.Duration // Refresh Token 过期时间
}

// NewJWTService 创建 JWT 服务实例。
//
// 参数：
//   - secret: HMAC-SHA256 签名密钥。生产环境必须通过环境变量注入，
//     不可使用默认值。长度建议至少 32 字节（256 位）。
//   - accessExpire: Access Token 过期时间，建议 15 分钟。
//   - refreshExpire: Refresh Token 过期时间，建议 7 天（168 小时）。
//
// secret 以字符串形式传入，内部转为 []byte 存储。
// 为什么外部用 string、内部用 []byte：secret 作为配置项是 string 类型，
// 但 golang-jwt 的 SignedString 需要 []byte，转换放在构造函数中更合理。
func NewJWTService(secret string, accessExpire, refreshExpire time.Duration) JWTService {
	return &jwtService{
		secret:        []byte(secret),
		accessExpire:  accessExpire,
		refreshExpire: refreshExpire,
	}
}

// GenerateAccessToken 生成访问令牌（Access Token）。
//
// 令牌内容（Payload 中的自定义字段）：
//   - user_id: 用户 ID
//   - username: 用户名
//   - role: 角色
//
// 标准字段：
//   - exp: 过期时间 = 当前时间 + accessExpire
//   - iat: 签发时间 = 当前时间
//   - nbf: 生效时间 = 当前时间（立即生效）
//
// 签名算法：HMAC-SHA256（jwt.SigningMethodHS256）。
// 签名使用配置的 secret 密钥。
//
// 返回值：
//   - string: 完整的 JWT 字符串（格式：header.payload.signature）。
//   - error: HMAC 签名失败时返回错误（通常不会发生）。
func (s *jwtService) GenerateAccessToken(userID int64, username, role string) (string, error) {
	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken 生成刷新令牌（Refresh Token）。
//
// 与 Access Token 的唯一区别是过期时间更长（refreshExpire）。
// 数据结构完全相同（包含 user_id, username, role），
// 这样 ValidateToken 可以统一验证两种令牌。
//
// 注意：当前实现中 Access Token 和 Refresh Token 在数据结构上
// 没有任何区分标记。如果将来需要区分（例如不允许用 Access Token
// 来刷新），可以在 CustomClaims 中添加一个 Type 字段。
func (s *jwtService) GenerateRefreshToken(userID int64, username, role string) (string, error) {
	claims := &CustomClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpire)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// ValidateToken 验证 JWT 令牌的有效性，返回解析后的 CustomClaims。
//
// 验证流程：
//  1. 使用 jwt.ParseWithClaims 解析 token。
//  2. 在 KeyFunc 中检查签名方法是否为 HMAC，防止攻击者
//     使用其他签名算法（如 "none" 算法）绕过验证。
//  3. 自动校验：签名、过期时间（exp）、生效时间（nbf）。
//  4. 解析 claims 类型为 CustomClaims。
//
// 返回的错误类型（供调用方判断用）：
//   - "token expired":       令牌已过期（exp < now）。
//   - "invalid token signature": 签名验证失败（secret 不匹配或 token 被篡改）。
//   - "invalid token":       其他解析错误（格式错误、算法不对等）。
//   - "invalid token claims": claims 类型断言失败。
//
// 安全性注意：
//   - golang-jwt 库默认会拒绝接受使用 "none" 算法的 token。
//   - KeyFunc 中额外检查了 token.Method 是否为 *jwt.SigningMethodHMAC，
//     增加一层保护。
func (s *jwtService) ValidateToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token expired")
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, errors.New("invalid token signature")
		}
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// RefreshAccessToken 使用刷新令牌获取新的访问令牌。
//
// 流程：
//  1. 验证传入的 refreshTokenString 是否有效（调用 ValidateToken）。
//  2. 如果有效，从 claims 中提取 user_id, username, role。
//  3. 使用相同信息签一个新的 Access Token。
//
// 注意：当前实现不会签发新的 Refresh Token。
// 这意味着 Refresh Token 在 7 天有效期内可以被重复使用。
// 如果希望实现 Refresh Token Rotation（每次刷新同时发新的
// refresh token，旧的失效），可以扩展此函数。
//
// 另外，当前实现没有区分传入的是 Access Token 还是 Refresh Token。
// 任何有效的 JWT 都可以用来换取新的 Access Token。
// 如果需要约束只允许 Refresh Token，可以在 CustomClaims 中
// 增加 Type 字段做区分。
func (s *jwtService) RefreshAccessToken(refreshTokenString string) (string, error) {
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", err
	}
	return s.GenerateAccessToken(claims.UserID, claims.Username, claims.Role)
}
