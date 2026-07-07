package handler

import (
	"net/http"
	"strconv"

	"backend/internal/service"
	"backend/pkg/errors"
	"backend/pkg/excel"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

// ImportExportHandler 处理 Excel 导入导出请求。
//
// 导出的数据流：DAO 查询 → service 转为 [][]string → excel.WriteSheet → []byte → HTTP 响应
// 导入的数据流：HTTP multipart 文件 → excel.ReadSheet → [][]string → service 解析并写入 DAO
type ImportExportHandler struct {
	svc service.ImportExportService
}

// NewImportExportHandler 创建导入导出处理器
func NewImportExportHandler(svc service.ImportExportService) *ImportExportHandler {
	return &ImportExportHandler{svc: svc}
}

// ExportPorts 导出所有港口为 Excel 文件下载。
// 文件头：Content-Disposition: attachment; filename=ports.xlsx
func (h *ImportExportHandler) ExportPorts(c *gin.Context) {
	data, err := h.svc.ExportPorts(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=ports.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportPorts 上传 Excel 文件批量导入港口。
// 使用 multipart/form-data 格式，字段名 file。
func (h *ImportExportHandler) ImportPorts(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c.Writer, "missing file")
		return
	}
	defer file.Close()

	rows, err := excel.ReadSheet(file, header.Size)
	if err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	imported, err := h.svc.ImportPorts(c.Request.Context(), rows)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"imported": imported})
}

// ExportVessels 导出所有船舶为 Excel。
func (h *ImportExportHandler) ExportVessels(c *gin.Context) {
	data, err := h.svc.ExportVessels(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=vessels.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportVessels 上传 Excel 批量导入船舶。
func (h *ImportExportHandler) ImportVessels(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c.Writer, "missing file")
		return
	}
	defer file.Close()

	rows, err := excel.ReadSheet(file, header.Size)
	if err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	imported, err := h.svc.ImportVessels(c.Request.Context(), rows)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"imported": imported})
}

// ExportShippingLines 导出所有航线为 Excel。
func (h *ImportExportHandler) ExportShippingLines(c *gin.Context) {
	data, err := h.svc.ExportShippingLines(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=shipping_lines.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportShippingLines 上传 Excel 批量导入航线。
func (h *ImportExportHandler) ImportShippingLines(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c.Writer, "missing file")
		return
	}
	defer file.Close()

	rows, err := excel.ReadSheet(file, header.Size)
	if err != nil {
		response.BadRequest(c.Writer, err.Error())
		return
	}
	imported, err := h.svc.ImportShippingLines(c.Request.Context(), rows)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	response.Success(c.Writer, gin.H{"imported": imported})
}

// ExportOrders 根据 shipper_company_id 导出订单为 Excel。
// 权限：shipping 角色不能使用此接口（订单按货主导出，与船公司无关）。
func (h *ImportExportHandler) ExportOrders(c *gin.Context) {
	role, _ := c.Get("role")
	if role == "shipping" {
		response.ErrorWithCode(c.Writer, errors.CodeForbidden, "shipping company cannot export by shipper")
		return
	}

	if role == "shipper" {
		userID, _ := c.Get("user_id")
		uid, ok := userID.(int64)
		if !ok {
			response.ErrorWithCode(c.Writer, errors.CodeForbidden, "invalid user identity")
			return
		}
		shipperIDStr := c.Query("shipper_company_id")
		if shipperIDStr != "" {
			shipperID, err := strconv.ParseInt(shipperIDStr, 10, 64)
			if err == nil && shipperID != uid {
				response.ErrorWithCode(c.Writer, errors.CodeForbidden, "can only export your own orders")
				return
			}
		}
		shipperIDStr = c.Query("shipper_company_id")
		if shipperIDStr == "" {
			shipperIDStr = strconv.FormatInt(uid, 10)
		}
		shipperID, _ := strconv.ParseInt(shipperIDStr, 10, 64)
		data, err := h.svc.ExportOrders(c.Request.Context(), shipperID)
		if err != nil {
			response.InternalServerError(c.Writer, err.Error())
			return
		}
		c.Header("Content-Disposition", "attachment; filename=orders.xlsx")
		c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
		return
	}

	shipperIDStr := c.Query("shipper_company_id")
	if shipperIDStr == "" {
		response.BadRequest(c.Writer, "shipper_company_id is required")
		return
	}
	shipperID, err := strconv.ParseInt(shipperIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid shipper_company_id")
		return
	}
	data, err := h.svc.ExportOrders(c.Request.Context(), shipperID)
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=orders.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}
