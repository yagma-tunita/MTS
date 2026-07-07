// Package idgen 提供分布式唯一 ID 生成功能。
//
// 算法选择：索尼的 Sonyflake（Snowflake 变种）。
//
// 为什么不用自增 ID（AUTO_INCREMENT）：
//   - 自增 ID 依赖数据库，在分库分表或读写分离场景下无法保证全局唯一。
//   - 自增 ID 是顺序的，泄露了系统中有多少条记录（安全风险）。
//
// 为什么不用 UUID：
//   - UUID 是 128 位字符串（36 字符），占 BIGINT 两倍的存储空间。
//   - UUID 无序，作为 InnoDB 主键时导致 B+ 树频繁页分裂，插入性能差。
//
// Sonyflake ID 的 64 位结构：
//   ┌──┬───────────────┬──────────┬──────────────┐
//   │0 │  39 bits      │ 8 bits   │ 16 bits      │
//   │r │  timestamp    │ sequence │ machine ID   │
//   └──┴───────────────┴──────────┴──────────────┘
//   - 1 bit 未使用（符号位，始终为 0，保证 ID 为正数）
//   - 39 bit 时间戳（毫秒级，从自定义起始时间开始算，可用 174 年）
//   - 8 bit 序列号（同一毫秒内可生成 256 个 ID）
//   - 16 bit 机器 ID（默认取 IPv4 地址的低 16 位）
//
// 机器 ID 的自动推导：
//   Sonyflake 默认使用机器私有 IP 的最后两个字节作为 machineID。
//   在单机部署中，machineID 固定不变；在多实例部署中，只要各实例
//   IP 不同，生成的 ID 就不会冲突。如果运行在容器中（所有实例
//   共享同一 IP），需要显式设置 Settings.MachineID 为不同的值。
//
// 使用方式：直接调用 idgen.NextID()，无需初始化。
// sync.Once 确保生成器只创建一次。
package idgen

import (
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

var (
	sf   *sonyflake.Sonyflake // Sonyflake 实例，延迟初始化
	once sync.Once            // 确保 initSnowflake 只执行一次
)

// initSnowflake 初始化 Sonyflake 实例。
//
// StartTime 设置为 2024-01-01 00:00:00 UTC：
//   这是 ID 时间戳的基准点。时间戳存储的是从 StartTime 到现在的毫秒数。
//   选择 2024 年是因为这个项目在 2024 年开始开发。
//   39 位时间戳最大可以表示约 174 年（2^39 / 1000 / 3600 / 24 / 365），
//   从 2024 开始，到 2198 年才会用完，完全够用。
//
// 如果 NewSonyflake 返回 nil（配置错误），直接 panic 终止程序，
// 因为 ID 生成器不可用会影响整个系统的正常运行。
func initSnowflake() {
	once.Do(func() {
		st := sonyflake.Settings{
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		sf = sonyflake.NewSonyflake(st)
		if sf == nil {
			panic("sonyflake not created")
		}
	})
}

// NextID 生成一个全局唯一的 uint64 类型 ID。
//
// 正常情况：返回 Sonyflake 生成的 64 位 ID，按时间递增。
// 异常降级：如果 Sonyflake 生成失败（如系统时钟回拨），
//   降级使用当前纳秒时间戳作为 ID。
//   时钟回拨通常发生在 NTP 时间同步时，概率很低。
//   降级后的 ID 不保证唯一（纳秒级精度但不同实例可能重复），
//   但相比系统崩溃不可用，这是一个可接受的权衡。
//
// 线程安全：Sonyflake.NextID() 内部使用原子操作，无需外部加锁。
//
// 使用示例：
//
//	orderID := idgen.NextID()
//	order.OrderNo = fmt.Sprintf("ORD%d", orderID)
func NextID() uint64 {
	initSnowflake()
	id, err := sf.NextID()
	if err != nil {
		return uint64(time.Now().UnixNano())
	}
	return id
}
