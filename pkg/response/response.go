package response

import (
	"encoding/json"
	"net/http"

	pkgerr "backend/pkg/errors"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PageResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    PageMeta    `json:"meta"`
}

func JSON(w http.ResponseWriter, statusCode int, resp interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func Success(w http.ResponseWriter, data interface{}) {
	JSON(w, http.StatusOK, Response{
		Code:    pkgerr.CodeSuccess,
		Message: "success",
		Data:    data,
	})
}

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

func Error(w http.ResponseWriter, code int, message string) {
	JSON(w, http.StatusOK, Response{
		Code:    code,
		Message: message,
	})
}

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

func BadRequest(w http.ResponseWriter, message string) {
	JSON(w, http.StatusBadRequest, Response{
		Code:    pkgerr.CodeBadRequest,
		Message: message,
	})
}

func Unauthorized(w http.ResponseWriter, message string) {
	JSON(w, http.StatusUnauthorized, Response{
		Code:    pkgerr.CodeUnauthorized,
		Message: message,
	})
}

func Forbidden(w http.ResponseWriter, message string) {
	JSON(w, http.StatusForbidden, Response{
		Code:    pkgerr.CodeForbidden,
		Message: message,
	})
}

func NotFound(w http.ResponseWriter, message string) {
	JSON(w, http.StatusNotFound, Response{
		Code:    pkgerr.CodeNotFound,
		Message: message,
	})
}

func Conflict(w http.ResponseWriter, message string) {
	JSON(w, http.StatusConflict, Response{
		Code:    pkgerr.CodeConflict,
		Message: message,
	})
}

func TooManyRequests(w http.ResponseWriter, message string) {
	JSON(w, http.StatusTooManyRequests, Response{
		Code:    pkgerr.CodeTooManyRequests,
		Message: message,
	})
}

func InternalServerError(w http.ResponseWriter, message string) {
	JSON(w, http.StatusInternalServerError, Response{
		Code:    pkgerr.CodeInternal,
		Message: message,
	})
}
