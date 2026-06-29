package service

import (
	"fmt"

	"gorm.io/gorm"
)

// PageRequest holds pagination and sorting parameters.
type PageRequest struct {
	Page        int
	PageSize    int
	SortBy      string
	SortOrder   string
	AllowedSort map[string]bool
}

// DefaultOrderSortFields returns default sortable fields for orders.
func DefaultOrderSortFields() map[string]bool {
	return map[string]bool{
		"create_time":      true,
		"order_no":         true,
		"total_weight_ton": true,
		"order_status":     true,
	}
}

// Paginate applies pagination and sorting to a GORM query.
// Returns the modified query, total count, and error.
func Paginate(db *gorm.DB, req PageRequest, model interface{}) (*gorm.DB, int64, error) {
	var total int64
	if err := db.Model(model).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Validate sorting
	sortBy := req.SortBy
	if sortBy == "" || (req.AllowedSort != nil && !req.AllowedSort[sortBy]) {
		sortBy = "create_time"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)

	// Validate pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	query := db.Order(orderClause).Offset(offset).Limit(pageSize)
	return query, total, nil
}
