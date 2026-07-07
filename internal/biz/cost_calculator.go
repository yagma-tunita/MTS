package biz

// CargoItem 货物条目，用于成本计算和运力校验。
// 这些字段在 handler 层从 JSON 解析，在 service 层转换为 biz.CargoItem。
type CargoItem struct {
	WeightTon float64 // 重量（吨），用于运力计算和运费计算
	VolumeM3  float64 // 体积（立方米），用于体积计算
	UnitPrice float64 // 单价（元/单位）
	Quantity  float64 // 数量
}

// CostResult 成本计算结果。
//
// ItemsSubtotal 的索引与传入的 CargoItem 切片索引一一对应，
// 方便调用方定位每个货物的小计金额。
type CostResult struct {
	TotalWeightTon float64   // 总重量（所有货物的 weight_ton 之和）
	TotalVolumeM3  float64   // 总体积（所有货物的 volume_cub_m 之和）
	TotalCost      float64   // 总费用（全部 subtotal 之和）
	ItemsSubtotal  []float64 // 每个货物的小计金额
}

// CostCalculator 成本计算器接口。
//
// 职责：将多个 CargoItem 汇总，计算总重量、总体积、总费用。
// 注意：运费（总费用 × 距离 × 系数）在此处不计算，
// 运费在 service 层组合了总重量、总航程、费率、货物系数后计算。
type CostCalculator interface {
	Calculate(items []CargoItem) (*CostResult, error)
}

// costCalculator 是 CostCalculator 接口的私有实现。
type costCalculator struct{}

// NewCostCalculator 创建成本计算器实例。
func NewCostCalculator() CostCalculator {
	return &costCalculator{}
}

// Calculate 计算货物汇总数据。
//
// 计算逻辑：
//   subtotal = quantity × unit_price
//   TotalWeightTon += weight_ton
//   TotalVolumeM3 += volume_cub_m
//   TotalCost += subtotal
//
// 返回非 nil error 的唯一条件是 items 为空（ErrEmptyCargoList）。
func (c *costCalculator) Calculate(items []CargoItem) (*CostResult, error) {
	if len(items) == 0 {
		return nil, ErrEmptyCargoList
	}
	result := &CostResult{
		ItemsSubtotal: make([]float64, len(items)),
	}
	for i, it := range items {
		subtotal := it.Quantity * it.UnitPrice
		result.TotalWeightTon += it.WeightTon
		result.TotalVolumeM3 += it.VolumeM3
		result.TotalCost += subtotal
		result.ItemsSubtotal[i] = subtotal
	}
	return result, nil
}
