package biz

// SegmentCalculator 航段计算器。
//
// 根据港口序列和起止港口，计算货物运输途经的所有邻接航段。
// 邻接航段 = 港口序列中相邻的两个港口（A→B, B→C, ...）。
//
// 为什么需要将订单拆分为航段：
//   运力是按航段管理的。货物从上海到鹿特丹，经过上海→新加坡→鹿特丹，
//   分别占用"上海→新加坡"和"新加坡→鹿特丹"两个航段的容量。
//   只有按航段管理，才能精确计算不同订单在同一航段上的容量竞争。
//   （详细说明见 QA.md Q5）
//
// 例如：
//   港口序列 [1, 2, 3, 5, 7]
//   起点 2, 终点 5
//   返回 [(2, 3), (3, 5)] —— 两个邻接航段
type SegmentCalculator interface {
	Calculate(portIDs []int64, startPortID, endPortID int64) ([][2]int64, error)
}

// segmentCalculator 是 SegmentCalculator 接口的私有实现。
type segmentCalculator struct{}

// NewSegmentCalculator 创建航段计算器实例。
func NewSegmentCalculator() SegmentCalculator {
	return &segmentCalculator{}
}

// Calculate 计算从 startPortID 到 endPortID 途径的所有邻接航段。
//
// 算法：
//  1. 在 portIDs 中查找 startPortID 和 endPortID 的索引。
//  2. 如果任一未找到，返回 ErrPortNotFoundInSeq。
//  3. 如果 start 索引 >= end 索引，返回 ErrStartAfterEnd（航线不可逆）。
//  4. 从 start 索引遍历到 end-1，每对相邻港口组成一个航段。
//
// 时间复杂度 O(n)，n 为港口序列长度（通常 < 10，效率无关紧要）。
func (c *segmentCalculator) Calculate(portIDs []int64, startPortID, endPortID int64) ([][2]int64, error) {
	startIdx, endIdx := -1, -1
	for i, pid := range portIDs {
		if pid == startPortID {
			startIdx = i
		}
		if pid == endPortID {
			endIdx = i
		}
	}
	if startIdx == -1 || endIdx == -1 {
		return nil, ErrPortNotFoundInSeq
	}
	if startIdx >= endIdx {
		return nil, ErrStartAfterEnd
	}
	segments := make([][2]int64, 0, endIdx-startIdx)
	for i := startIdx; i < endIdx; i++ {
		segments = append(segments, [2]int64{portIDs[i], portIDs[i+1]})
	}
	return segments, nil
}

