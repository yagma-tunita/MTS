package service

import (
	"fmt"

	"gorm.io/gorm"
)

// PageRequest 分页查询请求参数。
//
// 允许前端按照不同字段排序。为了安全，通过 AllowedSort 字段
// 限制前端只能排序在白名单中的字段，防止 SQL 注入（虽然 ORM
// 已经做了参数化查询，但排序字段是拼接的，需白名单保护）。
type PageRequest struct {
	Page        int               // 页码（从 1 开始）
	PageSize    int               // 每页条数
	SortBy      string            // 排序字段名
	SortOrder   string            // 排序方向：asc / desc
	AllowedSort map[string]bool   // 允许排序的字段白名单
}

// DefaultOrderSortFields 返回订单列表默认允许排序的字段白名单。
//
// 允许排序的字段需要满足两个条件：
//  1. 是 shipping_order 表中的真实列名。
//  2. 前端确实有按这个字段排序的需求。
func DefaultOrderSortFields() map[string]bool {
	return map[string]bool{
		"create_time":      true, // 创建时间（默认）
		"order_no":         true, // 订单号
		"total_weight_ton": true, // 总重量
		"order_status":     true, // 订单状态
	}
}

// Paginate 对 GORM 查询应用分页和排序。
//
// 流程：
//  1. 统计满足条件的总记录数（用于计算总页数）。
//  2. 校验排序字段合法性（防止 SQL 注入）。
//  3. 校验分页参数合法性（page 最小 1，pageSize 范围 1-100）。
//  4. 组装 ORDER BY + LIMIT + OFFSET 查询。
//
// 返回值：
//   - *gorm.DB: 已应用排序和分页的查询，可继续链式调用。
//   - int64: 总记录数（不分页）。
//   - error: 统计查询失败时返回错误。
//
// 使用示例：
//
//	query := db.Model(&model.ShippingOrder{}).Where("shipper_company_id = ?", id)
//	paginatedQuery, total, err := Paginate(query, req, &model.ShippingOrder{})
//	var orders []model.ShippingOrder
//	paginatedQuery.Preload("City").Find(&orders)
func Paginate(db *gorm.DB, req PageRequest, model interface{}) (*gorm.DB, int64, error) {
	var total int64
	if err := db.Model(model).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	sortBy := req.SortBy
	if sortBy == "" || (req.AllowedSort != nil && !req.AllowedSort[sortBy]) {
		sortBy = "create_time"
	}
	sortOrder := req.SortOrder
	if sortOrder != "asc" && sortOrder != "desc" {
		sortOrder = "desc"
	}
	if db.Statement.Table != "" && sortBy != "" {
		sortBy = db.Statement.Table + "." + sortBy
	}
	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)

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
