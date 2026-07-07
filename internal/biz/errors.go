// Package biz 实现核心业务逻辑（领域层）。
//
// 设计目标："零依赖"——不依赖任何框架、数据库、HTTP 或外部 I/O。
// biz 层是纯粹的 Go 函数/结构体，只做输入到输出的计算。
//
// 为什么 biz 层不能依赖任何 I/O：
//   业务规则（如"航段容量是否足够"）本身不关心数据从哪里来。
//   数据从 DAO 来还是从 Redis 来，或者从文件来，都不影响
//   业务规则的计算结果。通过"依赖倒置"——biz 定义接口，
//   service 实现数据获取回调——biz 层保持纯净。
//
// 当前 biz 层包含的模块：
//   1. PortSequenceParser  — 解析 JSON 格式的港口序列
//   2. SegmentCalculator   — 根据起止港计算航经航段
//   3. CapacityChecker     — 校验所有航段容量是否足够
//   4. CostCalculator      — 计算货物汇总（重量、体积、费用）
//   5. OrderNoGenerator    — 生成唯一订单号
//   6. OrderStateMachine   — 订单状态转换规则
//   7. VoyageRecommender   — 推荐可用航次
package biz

import "errors"

// ── 业务错误（biz 层定义的错误，service 层通过 errors.Is 判断） ──

// 这些错误在 biz 层内部使用。service 层通过 errors.Is 或直接
// 比较判断错误类型，然后转换为 pkg/errors.AppError 返回给 handler。
var (
	ErrInvalidPortSequence    = errors.New("invalid port sequence")              // 港口序列 JSON 解析失败或格式异常
	ErrPortNotFoundInSeq      = errors.New("start or end port not found in sequence") // 起止港口不在航线中
	ErrStartAfterEnd          = errors.New("start port appears after end port")  // 起港在止港之后（物理上不可逆）
	ErrInsufficientCapacity   = errors.New("insufficient capacity on segment")   // 航段剩余容量不足
	ErrInvalidStateTransition = errors.New("invalid order state transition")     // 订单状态转换不合法
	ErrInvalidOrderNoFormat   = errors.New("invalid order number format")        // 订单号格式错误
	ErrEmptyCargoList         = errors.New("cargo list cannot be empty")         // 货物列表为空
)
