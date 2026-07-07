package handler

import (
	"strconv"

	"backend/internal/model"
	"backend/internal/service"
	"backend/pkg/response"

	"github.com/gin-gonic/gin"
)

type CityHandler struct {
	svc service.CityService
}

func NewCityHandler(svc service.CityService) *CityHandler {
	return &CityHandler{svc: svc}
}

func (h *CityHandler) ListCities(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 { pageSize = 20 }
	if pageSize > 100 { pageSize = 100 }
	cityName := c.Query("city_name")
	cities, total, err := h.svc.ListCities(c.Request.Context(), page, pageSize, cityName)
	if err != nil {
		response.InternalServerError(c.Writer, "failed to list cities")
		return
	}
	response.SuccessPage(c.Writer, cities, page, pageSize, total)
}

func (h *CityHandler) CreateCity(c *gin.Context) {
	var city model.City
	if err := c.ShouldBindJSON(&city); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	if err := h.svc.CreateCity(c.Request.Context(), &city); err != nil {
		response.InternalServerError(c.Writer, "failed to create city")
		return
	}
	response.Success(c.Writer, city)
}

func (h *CityHandler) UpdateCity(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid city id")
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c.Writer, "invalid request body")
		return
	}
	city := &model.City{CityID: id}
	if v, ok := req["city_name"]; ok { city.CityName = v.(string) }
	if v, ok := req["country"]; ok { s := v.(string); city.Country = &s }
	if v, ok := req["country_code"]; ok { s := v.(string); city.CountryCode = &s }
	if v, ok := req["timezone"]; ok { s := v.(string); city.Timezone = &s }
	if err := h.svc.UpdateCity(c.Request.Context(), city); err != nil {
		response.InternalServerError(c.Writer, "failed to update city")
		return
	}
	response.Success(c.Writer, city)
}

func (h *CityHandler) DeleteCity(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c.Writer, "invalid city id")
		return
	}
	if err := h.svc.DeleteCity(c.Request.Context(), id); err != nil {
		response.InternalServerError(c.Writer, "failed to delete city")
		return
	}
	response.Success(c.Writer, gin.H{"message": "city deleted"})
}
