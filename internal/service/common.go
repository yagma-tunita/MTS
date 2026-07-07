// Package service 实现业务编排层（Orchestration Layer）。
//
// 层职责：
//   - 组合多个 DAO 调用完成一个完整的业务操作
//   - 管理数据库事务（Begin/Commit/Rollback）
//   - 调用 biz 层进行纯业务规则计算
//   - 操作缓存（设置·失效）
//   - 记录业务操作日志
//
// service 与 handler 的区别：
//   handler 只做"请求→参数→响应"的格式转换，
//   service 做"具体要做什么"的业务决策。
//   例如创建订单时，handler 只负责解析请求参数，
//   service 负责：校验 cargo note 存在、计算运费、
//   检查容量、启动事务、写入多表、清除缓存。
//
// service 与 biz 的区别：
//   biz 是纯计算（输入→输出），不依赖任何 I/O（数据库、网络）。
//   service 编排业务流程，调用 I/O 操作。
//   例如容量检查：biz.CapacityChecker 只做数学比较，
//   service 负责从 DAO 获取已占用的吨位数据。
//
// 依赖关系：
//   service → dao（数据访问）+ biz（业务规则）+ cache（缓存）+ ws（推送）
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"backend/pkg/timeutil"

	"gorm.io/gorm"
)

// Logger service 层使用的全局日志器。
// 为什么不用 slog.Default()？
//   可以在这里添加 service 层的公共字段，如 "layer": "service"，
//   所有 service 的日志都会自动携带这个字段。
//   当前保持与 slog.Default 相同，后续需要时可以修改。
var Logger = slog.Default()

// ServiceError 是旧的错误类型，已废弃。
//
// 请使用 pkg/errors.AppError 替代。
// 保留此类型是为了兼容早期代码，新代码不应使用。
type ServiceError struct {
	Code    string
	Message string
}

// Error
func (e ServiceError) Error() string {
	return e.Message
}

// 以下为已废弃的错误码常量，仅保持向后兼容。
// 新代码应使用 pkg/errors 中定义的错误码。
const (
	ErrCodeOrderNotFound   = "ORDER_NOT_FOUND"
	ErrCodeInsufficientCap = "INSUFFICIENT_CAPACITY"
	ErrCodeLockFailed      = "LOCK_FAILED"
	ErrCodeInvalidPortSeq  = "INVALID_PORT_SEQUENCE"
	ErrCodeNoCargoNote     = "CARGO_NOTE_NOT_FOUND"
	ErrCodeVesselNotFound  = "VESSEL_NOT_FOUND"
	ErrCodeLineNotFound    = "LINE_NOT_FOUND"
)

// AcquireLock 使用 MySQL 的 GET_LOCK 函数获取命名锁。
//
// MySQL GET_LOCK 是咨询锁（Advisory Lock）：
//   - 不是表锁或行锁，而是业务逻辑层面的命名锁。
//   - 锁名是一个字符串，在 MySQL 实例级别唯一。
//   - 不同连接之间的命名锁互不干扰。
//   - 连接断开后自动释放（不会死锁）。
//
// 为什么用 GET_LOCK 而不是 SELECT FOR UPDATE？
//   FOR UPDATE 锁的是表中的行，而 GET_LOCK 可以锁任意"业务概念"。
//   在我们的场景中，一个"航次"（line_id + vessel_id + voyage_date）
//   不是一个具体的行，而是一组相关的行。GET_LOCK 可以锁定
//   这个虚拟的概念。
//
// 参数：
//   - tx: 当前事务的 GORM DB 实例。必须在事务中调用。
//   - lockName: 锁名称，建议格式 "voyage_{lineID}_{vesselID}_{date}"。
//   - timeoutSec: 获取锁的超时秒数。超时返回 false 而非阻塞等待。
//
// 返回值：
//   - bool: true 表示成功获取锁，false 表示超时被其他连接持有。
//   - error: SQL 执行错误。
//
// 注意事项：
//   - GET_LOCK 是 MySQL 特有的函数，不可移植到其他数据库。
//   - 必须在同一个连接中调用 RELEASE_LOCK 释放。
//   - 如果事务回滚，锁会被自动释放（MySQL >= 5.7.5）。
func AcquireLock(tx *gorm.DB, lockName string, timeoutSec int) (bool, error) {
	var result int
	err := tx.Raw("SELECT GET_LOCK(?, ?)", lockName, timeoutSec).Scan(&result).Error
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

// ReleaseLock 释放通过 GET_LOCK 获取的命名锁。
//
// 必须在获取锁的同一个数据库连接（同一个事务）中调用。
// 如果不释放，锁会一直保持直到连接断开或事务结束。
func ReleaseLock(tx *gorm.DB, lockName string) error {
	var result int
	return tx.Raw("SELECT RELEASE_LOCK(?)", lockName).Scan(&result).Error
}

// VoyageLockKey 生成航次锁的键名。
//
// 格式："voyage_{lineID}_{vesselID}_{date}"
// 这个命名空间保证了不同航次使用不同的锁名，
// 不会相互阻塞。
func VoyageLockKey(lineID, vesselID int64, voyageDate string) string {
	return fmt.Sprintf("voyage_%d_%d_%s", lineID, vesselID, voyageDate)
}

// MustParseDate 将 "YYYY-MM-DD" 格式字符串解析为 time.Time。
// 解析失败返回零值时间（无 panic），因为输入的日期已经在 handler
// 层通过 validator 的 date 规则校验过格式，这里应该总是成功的。
func MustParseDate(s string) time.Time {
	t, _ := timeutil.ParseDate(s)
	return t
}

// PtrInt8 返回 int8 变量的指针。
//
// GORM 中很多字段定义为 *int8（可空），创建模型实例时
// 需要取指针。这个函数简化了写法：
//
//	OrderStatus: PtrInt8(1)
//
// 而不是：
//
//	v := int8(1); OrderStatus: &v
func PtrInt8(v int8) *int8 {
	return &v
}

// WithTimeout 为 context 附加 30 秒超时。
//
// 用于 service 层的数据库操作，防止某个异常操作（如死锁）
// 无限阻塞 goroutine。30 秒是所有正常业务操作都能完成的
// 安全阈值。
//
// 注意：返回的 cancel 函数必须被调用，否则会导致 context 泄漏。
// 正确用法：
//
//	ctx, cancel := WithTimeout(parentCtx)
//	defer cancel()
func WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}

