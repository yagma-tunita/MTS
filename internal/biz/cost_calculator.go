package biz

type CargoItem struct {
	WeightTon float64
	VolumeM3  float64
	UnitPrice float64
	Quantity  float64
}

type CostResult struct {
	TotalWeightTon float64
	TotalVolumeM3  float64
	TotalCost      float64
	ItemsSubtotal  []float64
}

type CostCalculator interface {
	Calculate(items []CargoItem) (*CostResult, error)
}

type costCalculator struct{}

func NewCostCalculator() CostCalculator {
	return &costCalculator{}
}

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
