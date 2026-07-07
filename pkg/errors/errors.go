// Package errors 提供带错误码和调用堆栈的结构化业务错误。
//
// 为什么需要自定义错误类型而不是用标准 errors.New 或 fmt.Errorf：
//   - 标准库错误只是字符串，无法携带错误码。
//   - HTTP API 需要将不同的业务错误映射到不同的 HTTP 状态码。
//   - 调用栈信息有助于快速定位问题代码位置。
//
// 错误码编码规则：
//   ┌────────┬─────────────────────────────────────────────┐
//   │ 范围   │ 含义                                       │
//   ├────────┼─────────────────────────────────────────────┤
//   │ 0      │ 成功（非错误，用于 response 包）            │
//   │ 1000-1999 │ 客户端错误 —— 调用方的问题              │
//   │ 2000+  │ 服务端错误 —— 系统内部问题                 │
//   └────────┴─────────────────────────────────────────────┘
//
// 这种分类方式与 HTTP 状态码对应：
//   - 1000-1999 → 400 Bad Request / 401 Unauthorized / 403 Forbidden / 404 Not Found
//   - 2000+    → 500 Internal Server Error
//
// 使用方式：
//
//	// 创建简单错误
//	return pkgerr.NotFound("order not found")
//
//	// 包装下层错误
//	return pkgerr.Wrap(pkgerr.CodeDatabaseError, "failed to query order", originalErr)
//
//	// 在 handler 层判断
//	if appErr, ok := err.(*pkgerr.AppError); ok {
//	    response.ErrorWithCode(w, appErr.Code, appErr.Message)
//	}
package errors

import (
	"fmt"
	"runtime"
)

// AppError 是应用层结构化错误类型。
//
// 字段说明：
//   - Code:    业务错误码，用于前端判断错误类型（不要用 HTTP status code 判断业务错误）。
//   - Message: 面向用户的可读错误描述（英文，前端可展示）。
//   - Stack:   创建错误时的调用堆栈，每个元素格式为 "file:line function"。
//   - Cause:   被包装的原始 error（如数据库返回的 gorm.ErrRecordNotFound）。
//     通过 errors.Unwrap() 暴露，支持 errors.Is/As 链式解包。
//
// 堆栈捕获说明：
//   - 只有通过 New/Wrap 创建的 AppError 才带堆栈。
//   - 预定义的构造函数（BadRequest、NotFound 等）内部调用 New，也带堆栈。
//   - 堆栈深度限制为 32 帧，基本能覆盖整个调用链。
//   - 跳过 captureStack 自身和 New/Wrap 共 3 层，从业务调用处开始记录。
type AppError struct {
	Code    int       // 业务错误码
	Message string    // 可读错误消息
	Stack   []string   // 调用堆栈（仅用于调试和日志记录）
	Cause   error     // 被包装的原始错误
}

// Error 实现标准 error 接口。
//
// 格式：
//   - 无 Cause:  "[1003] order not found"
//   - 有 Cause:  "[2001] database error: record not found"
//
// 这个格式在日志中便于 grep 搜索，code 带括号便于区分。
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 返回被包装的原始错误，用于 errors.Is/As 链式解包。
//
// 例如：
//
//	err := pkgerr.Wrap(pkgerr.CodeDatabaseError, "query failed", gorm.ErrRecordNotFound)
//	errors.Is(err, gorm.ErrRecordNotFound) // true
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New 创建一个不带 cause 的业务错误，并捕获当前调用堆栈。
//
// 适用于明确的业务错误（如"订单不存在"、"参数不合法"），不涉及
// 下层系统错误。堆栈从调用 New 的位置开始记录。
//
// 参数：
//   - code:    业务错误码，使用预定义的 CodeXxx 常量。
//   - message: 错误描述，建议使用一致的英文短语。
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Stack:   captureStack(),
	}
}

// Wrap 创建一个包装了下层错误的业务错误，同时捕获调用堆栈。
//
// 适用于包装来自 DAO 或第三方库的错误：
//
//	_, err := s.dao.GetByID(id)
//	if err != nil {
//	    return pkgerr.Wrap(pkgerr.CodeDatabaseError, "query port failed", err)
//	}
//
// 参数：
//   - code:    业务错误码。
//   - message: 描述当前操作的上下文信息。
//   - cause:   被包装的原始 error。
func Wrap(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Stack:   captureStack(),
	}
}

// captureStack 捕获当前 goroutine 的调用堆栈。
//
// 实现机制：
//  1. runtime.Callers(depth, pcs) 获取程序计数器（PC）值数组，
//     depth=3 表示跳过 captureStack、New/Wrap、调用者共 3 层，
//     从实际的业务代码处开始记录。
//  2. runtime.CallersFrames(pcs) 将 PC 转成可读的 Frame 信息。
//  3. 遍历所有 Frame，格式化为 "file:line function" 字符串。
//
// 为什么 depth=3：
//
//	帧 0: captureStack              ← 要跳过
//	帧 1: New 或 Wrap               ← 要跳过
//	帧 2: BadRequest 或 NotFound    ← 要跳过（如果是预定义函数调用的）
//	帧 3: 实际的业务代码处          ← 从这里开始记录
//
// 深度 32 帧：一般 Go 调用链很难超过 32 层，这个深度足够覆盖
// 所有的业务场景。如果将来出现更深的调用链，可以增加此值。
func captureStack() []string {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	stack := make([]string, 0, n)
	for {
		frame, more := frames.Next()
		stack = append(stack, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return stack
}

// ──────────────────────────────────────────────────────────────────────────────
// 预定义错误码和便捷构造函数
// ──────────────────────────────────────────────────────────────────────────────

const (
	CodeSuccess         = 0     // 成功（非错误）
	CodeBadRequest      = 1000  // 请求参数错误：必填字段缺失、格式错误、校验不通过
	CodeUnauthorized    = 1001  // 未认证：token 缺失、无效或过期
	CodeForbidden       = 1002  // 无权限：角色不匹配、越权操作
	CodeNotFound        = 1003  // 资源不存在：查询的订单/用户/港口等未找到
	CodeConflict        = 1004  // 资源冲突：运力不足、重复操作、状态冲突
	CodeTooManyRequests = 1005  // 请求频率超限：被限流器拦截
	CodeInternal        = 2000  // 服务器内部错误：意料之外的异常
	CodeDatabaseError   = 2001  // 数据库错误：连接失败、查询异常、事务冲突
	CodeDependencyError = 2002  // 外部依赖错误：第三方 API 调用失败
)

// BadRequest 创建 Code=1000 的错误。参数错误。
func BadRequest(msg string) *AppError      { return New(CodeBadRequest, msg) }

// Unauthorized 创建 Code=1001 的错误。认证失败。
func Unauthorized(msg string) *AppError    { return New(CodeUnauthorized, msg) }

// Forbidden 创建 Code=1002 的错误。无权访问。
func Forbidden(msg string) *AppError       { return New(CodeForbidden, msg) }

// NotFound 创建 Code=1003 的错误。资源不存在。
func NotFound(msg string) *AppError        { return New(CodeNotFound, msg) }

// Conflict 创建 Code=1004 的错误。资源冲突（如运力不足、订单已取消）。
func Conflict(msg string) *AppError        { return New(CodeConflict, msg) }

// TooManyRequests 创建 Code=1005 的错误。请求频率超限。
func TooManyRequests(msg string) *AppError { return New(CodeTooManyRequests, msg) }

// Internal 创建 Code=2000 的错误。服务器内部错误。
func Internal(msg string) *AppError        { return New(CodeInternal, msg) }

// DatabaseError 创建 Code=2001 的错误，自动包装原始 error。
// 参数 err 是来自 DAO 层或 GORM 的原始错误。
func DatabaseError(err error) *AppError    { return Wrap(CodeDatabaseError, "database error", err) }
