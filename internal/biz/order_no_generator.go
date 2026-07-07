package biz

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// OrderNoGenerator 订单号生成器接口。
//
// 生成的订单号格式：前缀（默认 ORD）+ YYYYMMDD + 8 位随机 hex。
// 示例：ORD20260706a3f2c1b0
//
// 唯一性保证：
//   - 8 位随机 hex = 32 位随机数 ≈ 42 亿种可能。
//   - 同一天内两个订单碰撞的概率极低（约 2^-32）。
//   - 即使碰撞，数据库的唯一索引 uk_orderno_delete 会阻止插入。
//
// 为什么不使用自增序列：
//   订单号对可读性有要求——看到 ORD20260706 就知道是 2026 年 7 月 6 日的订单。
//   自增序列无法携带日期信息。
type OrderNoGenerator interface {
	Generate() string
}

// orderNoGenerator 是 OrderNoGenerator 接口的私有实现。
type orderNoGenerator struct {
	prefix string
}

// NewOrderNoGenerator 创建订单号生成器。prefix 默认为 ORD。
func NewOrderNoGenerator(prefix string) OrderNoGenerator {
	if prefix == "" {
		prefix = "ORD"
	}
	return &orderNoGenerator{prefix: prefix}
}

// Generate 生成唯一订单号。
//
// 使用 crypto/rand 而不是 math/rand 生成随机部分：
//   虽然订单号不需要密码学安全的随机性，但 crypto/rand 在
//   Linux 和 Windows 上都有良好支持，且不会像 math/rand 那样
//   需要初始化种子。使用它没有额外的开销。
func (g *orderNoGenerator) Generate() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s%s%s", g.prefix, time.Now().Format("20060102"), hex.EncodeToString(b))
}
