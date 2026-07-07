// Package response 提供统一的 HTTP JSON 响应格式和便捷函数。
//
// 统一响应格式（所有 API 都遵循）：
//
//	成功（无分页）：
//	  HTTP 200  {"code": 0, "message": "success", "data": {...}}
//
//	成功（有分页）：
//	  HTTP 200  {"code": 0, "message": "success", "data": [...], "meta": {...}}
//
//	失败（业务错误）：
//	  HTTP 200  {"code": 1001, "message": "unauthorized"}         — Error() 函数
//	  或：
//	  HTTP 401  {"code": 1001, "message": "unauthorized"}         — ErrorWithCode() 函数
//
// 为什么提供两种错误响应风格：
//   - Error()：始终返回 HTTP 200，业务错误通过 code 字段区分。
//     适合全部使用统一 HTTP 状态码的 RPC 风格。
//   - ErrorWithCode()：根据错误码自动映射 HTTP 状态码。
//     适合 RESTful 风格，前端可同时依赖 HTTP 状态码和业务 code。
//   - 便捷函数（BadRequest, Unauthorized 等）：直接设置正确的
//     HTTP 状态码 + 业务错误码，语义最清晰。
//
// 分页响应设计：
//   - data 字段放列表数据，meta 字段放分页信息。
//   - 这样前端可以统一处理分页：不管 data 是什么类型，
//     meta 的结构始终不变。
//   - total_pages 在服务端计算，避免前端再做一次除法。
//
// 依赖关系：引用 pkg/errors 的错误码常量。
package response

import (
	"encoding/json"
	"net/http"

	pkgerr "backend/pkg/errors"
)

// Response 标准 JSON 响应结构体。
//
// code=0 表示成功，非零值表示业务错误码（定义见 pkg/errors）。
// message 是成功或错误的文字描述。
// data 携带响应数据，成功时有值，错误时为空（由 omitempty 控制不输出）。
type Response struct {
	Code    int         `json:"code"`             // 业务码：0=成功，非零=错误
	Message string      `json:"message"`          // 提示信息
	Data    interface{} `json:"data,omitempty"`   // 响应数据（成功时携带）
}

// PageMeta 分页元信息。
//
// 所有分页接口统一使用此结构，保证前端做分页组件时接口一致。
type PageMeta struct {
	Page       int   `json:"page"`        // 当前页码（从 1 开始）
	PageSize   int   `json:"page_size"`   // 每页条数
	Total      int64 `json:"total"`       // 总记录数
	TotalPages int   `json:"total_pages"` // 总页数（由服务端计算）
}

// PageResponse 带分页元信息的响应结构体。
//
// 与 Response 的区别：多了 meta 字段，data 没有 omitempty（列表可为空数组）。
type PageResponse struct {
	Code    int         `json:"code"`           // 业务码
	Message string      `json:"message"`        // 提示信息
	Data    interface{} `json:"data"`           // 列表数据
	Meta    PageMeta    `json:"meta"`           // 分页信息
}

// JSON 是核心写入函数，将响应内容序列化为 JSON 并写入 http.ResponseWriter。
//
// 流程：
//  1. 设置响应头 Content-Type: application/json。
//  2. 调用 w.WriteHeader(statusCode) 写入 HTTP 状态码。
//  3. 使用 json.NewEncoder(w).Encode(resp) 序列化并写入响应体。
//
// 为什么用 json.NewEncoder 而不是 json.Marshal + w.Write：
//   - json.NewEncoder 直接写入 stream，不需要中间 buffer，内存更省。
//   - json.NewEncoder 在遇到无法序列化的类型时会在编码过程中
//     返回错误，但 HTTP 响应头已经发出，无法再返回 500。
//     不过在本项目中，所有响应数据结构都是简单可控的，
//     基本不会出现序列化失败的情况。
func JSON(w http.ResponseWriter, statusCode int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// Success 返回 HTTP 200 的成功响应，带 data 数据。
//
// 示例输出：
//
//	HTTP 200
//	{"code": 0, "message": "success", "data": {"order_id": 1}}
func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Code:    pkgerr.CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

// SuccessPage 返回 HTTP 200 的成功分页响应。
//
// 自动计算总页数：total / pageSize 向上取整。
// 如果 total 为 0，totalPages 也为 0（不会出现 0/0 问题，因为 Go 整数除法中 0/10=0）。
//
// 示例输出：
//
//	HTTP 200
//	{"code": 0, "message": "success", "data": [...], "meta": {"page": 1, "page_size": 20, "total": 50, "total_pages": 3}}
func SuccessPage(w http.ResponseWriter, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}
	JSON(w, http.StatusOK, PageResponse{
		Code:    pkgerr.CodeSuccess,
		Message: "success",
		Data:    data,
		Meta: PageMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// Error 返回 HTTP 200 + 业务错误码的响应（RPC 风格）。
//
// 示例输出：
//
//	HTTP 200
//	{"code": 1001, "message": "unauthorized"}
//
// 与 ErrorWithCode 的区别：Error 始终返回 200，
// 前端只能通过 code 字段判断错误类型。
// 适用于网关统一处理响应的场景。
func Error(w http.ResponseWriter, code int, message string) {
	JSON(w, http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

// ErrorWithCode 根据业务错误码自动映射 HTTP 状态码（REST 风格）。
//
// 映射规则：
//
//	业务码                  → HTTP 状态码
//	CodeUnauthorized(1001)  → 401 Unauthorized
//	CodeForbidden(1002)     → 403 Forbidden
//	CodeNotFound(1003)      → 404 Not Found
//	CodeTooManyRequests(1005) → 429 Too Many Requests
//	1000-1999（其他客户端错误）→ 400 Bad Request
//	2000+（服务端错误）     → 500 Internal Server Error
//	其他                    → 200 OK
//
// 为什么 1000-1999 映射到 400 而不是各自不同的 HTTP 状态码：
//   1000-1999 范围内的错误码对应的是"客户端的问题"，但具体语义
//   比 HTTP 的 400/401/403/404 更细致。HTTP 只有约 10 个
//   客户端错误状态码，而业务错误码可能有几十个。映射到 400
//   是"都是调用方的锅"这个层面的归类，前端通过 code 字段精确定位。
func ErrorWithCode(w http.ResponseWriter, code int, message string) {
	httpStatus := http.StatusOK
	switch {
	case code == pkgerr.CodeUnauthorized:
		httpStatus = http.StatusUnauthorized
	case code == pkgerr.CodeForbidden:
		httpStatus = http.StatusForbidden
	case code == pkgerr.CodeNotFound:
		httpStatus = http.StatusNotFound
	case code == pkgerr.CodeTooManyRequests:
		httpStatus = http.StatusTooManyRequests
	case code >= 1000 && code < 2000:
		httpStatus = http.StatusBadRequest
	case code >= 2000:
		httpStatus = http.StatusInternalServerError
	}
	JSON(w, httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

// 以下为便捷错误响应函数。
// 每个函数同时设置了正确的 HTTP 状态码和业务错误码，
// 在 handler 层使用最简洁。

// BadRequest 400 Bad Request。请求参数校验失败。
func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, Response{Code: pkgerr.CodeBadRequest, Message: message})
}

// Unauthorized 401 Unauthorized。Token 缺失或无效。
func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, Response{Code: pkgerr.CodeUnauthorized, Message: message})
}

// Forbidden 403 Forbidden。角色权限不足。
func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, Response{Code: pkgerr.CodeForbidden, Message: message})
}

// NotFound 404 Not Found。请求的资源不存在。
func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, Response{Code: pkgerr.CodeNotFound, Message: message})
}

// Conflict 409 Conflict。资源冲突（如运力不足、订单已取消）。
func Conflict(w http.ResponseWriter, message string) {
	JSON(w, http.StatusConflict, Response{Code: pkgerr.CodeConflict, Message: message})
}

// TooManyRequests 429 Too Many Requests。请求频率超限。
func TooManyRequests(w http.ResponseWriter, message string) {
	JSON(w, http.StatusTooManyRequests, Response{Code: pkgerr.CodeTooManyRequests, Message: message})
}

// InternalServerError 500 Internal Server Error。服务器内部错误。
func InternalServerError(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, Response{Code: pkgerr.CodeInternal, Message: message})
}
