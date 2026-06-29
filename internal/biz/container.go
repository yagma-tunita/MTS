package biz

// BizContainer holds all business components.
type BizContainer struct {
	PortSequenceParser PortSequenceParser
	SegmentCalculator  SegmentCalculator
	CapacityChecker    CapacityChecker
	OrderNoGenerator   OrderNoGenerator
	CostCalculator     CostCalculator
	OrderStateMachine  OrderStateMachine
	VoyageRecommender  VoyageRecommender
}

func NewBizContainer() *BizContainer {
	segCalc := NewSegmentCalculator()
	return &BizContainer{
		PortSequenceParser: NewPortSequenceParser(),
		SegmentCalculator:  segCalc,
		CapacityChecker:    NewCapacityChecker(),
		OrderNoGenerator:   NewOrderNoGenerator("ORD"),
		CostCalculator:     NewCostCalculator(),
		OrderStateMachine:  NewOrderStateMachine(),
		VoyageRecommender:  NewVoyageRecommender(segCalc),
	}
}
