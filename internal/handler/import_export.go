package handler

import (
	"net/http"
	"strconv"

	"backend/internal/service"
	"backend/pkg/excel"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type ImportExportHandler struct {
	svc service.ImportExportService
}

func NewImportExportHandler(svc service.ImportExportService) *ImportExportHandler {
	return &ImportExportHandler{svc: svc}
}

// ExportPorts godoc
// @Summary      Export ports to Excel
// @Description  Download all ports as an Excel file
// @Tags         Import/Export
// @Produce      application/octet-stream
// @Success      200 {file} file
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /export/ports [get]
func (h *ImportExportHandler) ExportPorts(c *gin.Context) {
	data, err := h.svc.ExportPorts(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=ports.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportPorts godoc
// @Summary      Import ports from Excel
// @Description  Upload an Excel file to add new ports
// @Tags         Import/Export
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Excel file"
// @Success      200 {object} response.Response{data=object{imported=int}}
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /import/ports [post]
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

// ExportVessels godoc
// @Summary      Export vessels to Excel
// @Description  Download all vessels as an Excel file
// @Tags         Import/Export
// @Produce      application/octet-stream
// @Success      200 {file} file
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /export/vessels [get]
func (h *ImportExportHandler) ExportVessels(c *gin.Context) {
	data, err := h.svc.ExportVessels(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=vessels.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportVessels godoc
// @Summary      Import vessels from Excel
// @Description  Upload an Excel file to add new vessels
// @Tags         Import/Export
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Excel file"
// @Success      200 {object} response.Response{data=object{imported=int}}
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /import/vessels [post]
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

// ExportShippingLines godoc
// @Summary      Export shipping lines to Excel
// @Description  Download all shipping lines as an Excel file
// @Tags         Import/Export
// @Produce      application/octet-stream
// @Success      200 {file} file
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /export/shipping-lines [get]
func (h *ImportExportHandler) ExportShippingLines(c *gin.Context) {
	data, err := h.svc.ExportShippingLines(c.Request.Context())
	if err != nil {
		response.InternalServerError(c.Writer, err.Error())
		return
	}
	c.Header("Content-Disposition", "attachment; filename=shipping_lines.xlsx")
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

// ImportShippingLines godoc
// @Summary      Import shipping lines from Excel
// @Description  Upload an Excel file to add new shipping lines
// @Tags         Import/Export
// @Accept       multipart/form-data
// @Produce      json
// @Param        file formData file true "Excel file"
// @Success      200 {object} response.Response{data=object{imported=int}}
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /import/shipping-lines [post]
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

// ExportOrders godoc
// @Summary      Export orders to Excel
// @Description  Download orders for a specific shipper as an Excel file
// @Tags         Import/Export
// @Produce      application/octet-stream
// @Param        shipper_company_id query int true "Shipper company ID"
// @Success      200 {file} file
// @Failure      400 {object} response.Response
// @Failure      500 {object} response.Response
// @Security     BearerAuth
// @Router       /export/orders [get]
func (h *ImportExportHandler) ExportOrders(c *gin.Context) {
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
