// Package cache 提供进程内内存缓存功能，基于 github.com/patrickmn/go-cache 实现。
//
// 设计目标：
//   - 为频繁读取但不常变更的数据（如港口列表、运力推荐结果）提供临时存储，
//     减少对数据库的直接查询压力。
//   - 所有操作通过包级函数完成，无需显式创建实例，开箱即用。
//
// 存储机制：
//   - 底层使用线程安全的 map[string]Item，每个 Item 包含值和过期时间。
//   - init() 时创建一个默认缓存实例（defaultCache），TTL=5分钟，清理间隔=10分钟。
//   - 后台 goroutine 每 10 分钟扫描一次过期条目并自动删除。
//
// 适用场景：
//   - 港口详情/列表缓存（TTL 10 分钟，数据极少变更）
//   - 运力推荐结果缓存（TTL 1 分钟，下单/取消时通过 DeletePrefix 主动失效）
//
// 注意事项：
//   - 缓存仅存在于当前进程内存中，服务重启后数据丢失。
//   - 多实例部署（水平扩展）时各实例缓存不一致，应替换为 Redis 等分布式缓存。
//   - DeletePrefix 采用全量遍历方式，缓存量大时性能不佳（当前业务场景可接受）。
package cache

import (
	"time"

	"github.com/patrickmn/go-cache"
)

// defaultCache 是包级别的全局缓存实例。
// 为什么是包级单例而不是实例化使用：简化调用方代码，无需传递缓存对象，
// 业务层只需 import "backend/pkg/cache" 后直接调用 cache.Get/cache.Set。
var defaultCache *cache.Cache

// init 在包加载时自动执行，创建默认缓存实例。
// cache.New 的两个参数：
//   - defaultExpiration: 5min —— 当 Set 未指定 TTL 时使用的默认过期时间。
//   - cleanupInterval: 10min —— 后台清理 goroutine 的运行间隔，
//     每隔 10 分钟扫描并删除已过期的键，释放内存。
//
// 为什么默认 TTL 设为 5 分钟：业务数据（港口、推荐结果）的变更频率较低，
// 5 分钟缓存可以在数据新鲜度和数据库压力之间取得平衡。
func init() {
	defaultCache = cache.New(5*time.Minute, 10*time.Minute)
}

// Set 存储一个键值对到缓存中，并指定 TTL（生存时间）。
//
// 参数：
//   - key: 缓存的键，建议使用有意义的命名空间格式，例如：
//     "port:id:123"（单个港口）、"ports:list:1:20"（港口列表）、
//     "voyage_rec:1:3:500.00"（运力推荐）。
//   - value: 任意类型的值。底层 go-cache 使用 interface{} 存储，
//     取回时调用方需自行做类型断言。
//   - ttl: 生存时间。如果 ttl <= 0 或未指定，使用默认值 5 分钟。
//
// 线程安全：go-cache 内部使用 sync.RWMutex 保护所有读写操作。
func Set(key string, value interface{}, ttl time.Duration) {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	defaultCache.Set(key, value, ttl)
}

// Get 从缓存中获取指定键的值。
//
// 返回值：
//   - interface{}: 存储的值。如果键不存在或已过期则为 nil。
//   - bool: true 表示命中缓存且值未过期，false 表示未命中。
//
// 使用示例：
//
//	if cached, found := cache.Get("port:id:123"); found {
//	    if port, ok := cached.(*model.Port); ok {
//	        return port, nil
//	    }
//	}
//
// 注意：必须对返回值进行类型断言，因为底层存储的是 interface{}。
func Get(key string) (interface{}, bool) {
	return defaultCache.Get(key)
}

// Delete 从缓存中删除单个键。键不存在时不会报错。
func Delete(key string) {
	defaultCache.Delete(key)
}

// DeletePrefix 删除所有以指定前缀开头的缓存键。
//
// 实现方式：遍历 defaultCache.Items() 获取所有键值对，
// 对每个键检查是否以 prefix 开头，匹配则删除。
// 时间复杂度 O(n)，n 为缓存中条目总数。
//
// 使用场景：当某个业务操作导致一批缓存失效时。
// 例如创建订单后调用 cache.DeletePrefix("voyage_rec:")，
// 清空所有运力推荐缓存，确保下次查询能获取到最新数据。
//
// 注意：如果缓存条目数量达到数万级别，此方法会成为性能瓶颈。
// 当前系统缓存数据量很小（港口几十条、推荐结果几十条），O(n) 遍历无压力。
// 如需支持大规模缓存，应改用其他方案（如维护键列表或使用 Redis 的 SCAN）。
func DeletePrefix(prefix string) {
	items := defaultCache.Items()
	for k := range items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			defaultCache.Delete(k)
		}
	}
}

// Flush 清空整个缓存。调用后所有键值对都被移除。
// 用于测试环境重置或系统管理操作。
func Flush() {
	defaultCache.Flush()
}
