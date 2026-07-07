package biz

// CapacityChecker 容量检查器。
//
// 职责：校验新订单的货物总重量是否能在航线的所有航段上存放。
// 对于每个航段：已占用 + 新货物 <= 最大载重。
//
// 为什么通过回调函数（occupiedGetter）而不是直接接受已占用数据：
//   保持 biz 层的纯净性。CapacityChecker 不需要知道数据从哪来，
//   它的职责仅仅是"给数据、算结果"。数据获取由 service 层负责。
type CapacityChecker interface {
	// Check 检查所有航段是否能容纳 totalWeight。
	//
	// 参数：
	//   - segments: 从 SegmentCalculator 计算得到的航段列表。
	//   - maxWeight: 船舶最大载重（DWT）。
	//   - occupiedGetter: 回调函数，返回指定航段已占用的吨位。
	//   - totalWeight: 本订单需要占用的吨位。
	//
	// 返回值：
	//   - bool: true=所有航段容量足够，false=至少一个航段超容。
	//   - float64: 所有航段中的最小剩余容量（负数表示超了多少吨）。
	//   - error: 回调函数返回的错误。
	Check(segments [][2]int64, maxWeight float64, occupiedGetter func(seg [2]int64) (float64, error), totalWeight float64) (bool, float64, error)
}

// capacityChecker 是 CapacityChecker 接口的私有实现。
type capacityChecker struct{}

// NewCapacityChecker 创建容量校验器实例
func NewCapacityChecker() CapacityChecker {
	return &capacityChecker{}
}

// Check 遍历所有航段，对每个航段计算剩余容量 = maxWeight - used - totalWeight。
//
// 瓶颈航段：返回值 minRemaining 是所有航段中的最小值。
// 这个值可用于"这个订单完成后还剩多少容量"的预估。
//
// 注意：occupiedGetter 获取的是该航段上所有已有订单的已占用吨位之和，
// 而不是单个订单的。由 service 层的 DAO 负责执行 SUM() 聚合。
func (c *capacityChecker) Check(segments [][2]int64, maxWeight float64, occupiedGetter func([2]int64) (float64, error), totalWeight float64) (bool, float64, error) {
	var minRemaining float64 = -1
	for _, seg := range segments {
		used, err := occupiedGetter(seg)
		if err != nil {
			return false, 0, err
		}
		remaining := maxWeight - used - totalWeight
		if remaining < 0 {
			return false, remaining, ErrInsufficientCapacity
		}
		if minRemaining == -1 || remaining < minRemaining {
			minRemaining = remaining
		}
	}
	return true, minRemaining, nil
}
