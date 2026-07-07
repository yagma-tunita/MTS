// Package crypto 提供密码学相关工具函数。
//
// 包含四个功能组：
//  1. 密码哈希（bcrypt）—— 安全的密码单向哈希，用于用户密码存储。
//  2. 对称加密（AES-256-GCM）—— 带认证的加密，用于敏感数据加密存储。
//  3. 安全随机字符串（crypto/rand）—— 用于生成令牌、密钥等。
//  4. MD5 摘要—— 仅用于校验和或确定性 ID 生成，不可用于安全场景。
//
// 设计原则：
//   - 优先使用现代安全的算法（bcrypt、AES-GCM）。
//   - 每个函数返回值都包含 error，方便调用方处理失败情况。
//   - 对关键参数（如 AES 密钥长度）做明确的输入验证。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// ══════════════════════════════════════════════════════════════════════════════
// 1. Password hashing (bcrypt)
// ══════════════════════════════════════════════════════════════════════════════
//
// 为什么选 bcrypt 而不是 SHA-256：
//   - bcrypt 是慢哈希算法（默认 cost=10 时约需 100ms），
//     可以显著增加暴力破解的成本。
//   - SHA-256 是快速哈希，用 GPU 每秒可以尝试数十亿次，不适合密码存储。
//   - bcrypt 内部自动集成随机盐值（salt），无需手动处理。
//
// 为什么选 bcrypt 而不是 argon2：
//   - argon2 更新更安全，但标准库不直接支持，需要额外依赖。
//   - bcrypt 通过 golang.org/x/crypto 提供，是 Go 生态的标准选择。
//   - 对于当前业务场景，bcrypt 的安全性已经足够。

// HashPassword 使用 bcrypt 算法对明文密码进行哈希。
//
// bcrypt.DefaultCost = 10，意味着 2^10 = 1024 轮迭代。
// 每轮迭代对密码和随机盐值进行 Blowfish 加密。
// 输出格式：$2a$10$[22 字节盐值][31 字节哈希]（共 60 字符）。
//
// 参数：
//   - password: 明文密码字符串。
//
// 返回值：
//   - string: 编码后的 bcrypt 哈希值（包含算法标识、cost、盐值和哈希）。
//   - error: 如果 bcrypt 库内部出错（如系统熵不足导致盐值生成失败）。
//
// 使用示例：
//
//	hash, err := crypto.HashPassword("myPassword123")
//	// hash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

// CheckPasswordHash 将明文密码与 bcrypt 哈希值进行比对。
//
// 内部原理：bcrypt.CompareHashAndPassword 会从哈希中提取 salt 和 cost，
// 重新计算哈希后与存储值进行常量时间比较（防止时序攻击）。
//
// 参数：
//   - password: 用户输入的明文密码。
//   - hash: 之前通过 HashPassword 生成的 bcrypt 哈希。
//
// 返回值：
//   - bool: true 表示密码匹配，false 表示不匹配。
//
// 注意：返回 false 时不暴露具体失败原因（如"密码错误" vs "用户不存在"），
// 防止攻击者通过错误信息枚举有效用户。
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ══════════════════════════════════════════════════════════════════════════════
// 2. Random secure string generator
// ══════════════════════════════════════════════════════════════════════════════
//
// 使用 crypto/rand（系统 CSPRNG）而非 math/rand：
//   - crypto/rand 读取系统的随机数池（Windows 的 CryptGenRandom、
//     Linux 的 /dev/urandom），提供密码学安全的随机性。
//   - math/rand 是伪随机数生成器，可预测，不适合用于生成令牌和密钥。

// GenerateRandomString 生成一个密码学安全的随机十六进制字符串。
//
// 参数 n 为底层随机字节数，输出字符串长度为 2n（因为 hex 编码每个字节变两个字符）。
// 例如 n=16 时输出 32 字符的 hex 字符串。
//
// 安全性：
//   - 使用 crypto/rand.Read，系统级安全随机源。
//   - 输出的每个字符是 0-9a-f，信息熵 = 4n 比特。
//   - n=32 时熵为 128 比特，对于令牌生成已足够安全。
//
// 用途：
//   - 生成 API 令牌
//   - 生成重置密码链接中的临时令牌
//   - 生成会话 ID
func GenerateRandomString(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// MustGenerateRandomString 是 GenerateRandomString 的便捷版本，失败时 panic。
// 适用于初始化阶段或已知不可能失败（如 n 是合法小正整数）的场景。
func MustGenerateRandomString(n int) string {
	s, err := GenerateRandomString(n)
	if err != nil {
		panic(err)
	}
	return s
}

// ══════════════════════════════════════════════════════════════════════════════
// 3. AES-GCM encryption
// ══════════════════════════════════════════════════════════════════════════════
//
// AES-256-GCM（Galois/Counter Mode）提供认证加密（AEAD）：
//   - 加密：使用 AES-256（密钥 32 字节）对明文进行 CTR 模式加密。
//   - 认证：GCM 模式附加 128 位认证标签，确保密文未被篡改。
//   - 完整性：密文中的任意比特错误都会被认证标签检查捕获。
//
// 输出格式：nonce（12 字节）|| ciphertext（长度=明文）|| tag（16 字节）
// 使用 gcm.Seal 方法将 nonce 前置到密文前，解密时再切分。
//
// 每次加密生成随机 nonce（通过 crypto/rand），确保同一密钥下同一明文
// 每次加密产生不同密文。

// AESEncrypt 使用 AES-256-GCM 对明文进行加密。
//
// 参数：
//   - key: 32 字节密钥（AES-256 要求）。如果长度不是 32 字节，返回错误。
//   - plaintext: 要加密的明文。
//
// 返回值：
//   - []byte: nonce || ciphertext（认证标签嵌在 GCM 密文中）。
//   - error: 密钥长度错误或系统熵不足时返回。
//
// 使用示例：
//
//	key := make([]byte, 32)        // 生产环境应从安全密钥管理系统获取
//	rand.Read(key)                 // 演示用
//	ciphertext, err := AESEncrypt(key, []byte("sensitive data"))
func AESEncrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize()) // GCM 推荐 nonce 大小为 12 字节
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	// Seal 将 nonce 前置到 ciphertext 前，以便解密时提取
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt 使用 AES-256-GCM 解密密文（需要与 AESEncrypt 使用相同的密钥）。
//
// 参数：
//   - key: 32 字节密钥，必须与加密时使用的密钥相同。
//   - ciphertext: AESEncrypt 的输出格式（nonce || ciphertext || tag）。
//
// 返回值：
//   - []byte: 解密后的明文。
//   - error: 密钥错误、密文格式错误或认证标签验证失败时返回。
//
// 安全性：
//   - 如果密文被篡改（包括 nonce 或加密数据），gcm.Open 会返回错误。
//   - 不要忽略此错误——它表明数据完整性已被破坏。
func AESDecrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes for AES-256")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}
	return plaintext, nil
}

// ══════════════════════════════════════════════════════════════════════════════
// 4. MD5 (for non-security purposes)
// ══════════════════════════════════════════════════════════════════════════════
//
// MD5 已不适用于安全场景（碰撞攻击已可行），这里仅用于：
//   - 文件校验和（确认文件传输完整性）
//   - 确定性 ID 生成（从内容生成固定长度的标识符）
//   - 非安全场景的哈希运算

// MD5Hex 计算输入数据的 MD5 哈希值，返回十六进制字符串。
// 警告：MD5 不安全，不应用于密码存储、数字签名或任何需要抗碰撞的场景。
func MD5Hex(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// MD5String 是 MD5Hex 的字符串版本便利函数，直接传入字符串。
func MD5String(s string) string {
	return MD5Hex([]byte(s))
}
