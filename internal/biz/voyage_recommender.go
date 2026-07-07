package biz

import "sort"

// VoyageInfo 航次候选信息。由 service 层从数据库中查询拼装。
//
// 注意：MaxWeight 是船舶的最大载重吨（DWT），不是剩余容量。
// 剩余容量需要在 Recommend 时计算（通过回调函数 getRemaining）。
type VoyageInfo struct {
	LineID     int64   // 航线 ID
	VesselID   int64   // 船舶 ID
	VoyageDate string  // 航次日期（YYYY-MM-DD）
	VesselName string  // 船舶名称（用于展示）
	LineName   string  // 航线名称（用于展示）
	MaxWeight  float64 // 船舶最大载重吨
	PortIDs    []int64 // 航线港口序列（已解析为 []int64）
}

// SegmentRemainingGetter 查询指定航段剩余容量的函数类型。
// service 层实现此函数，内部调用 DAO 从 segment_capacity_usage 表 SUM 已占用吨位。
type SegmentRemainingGetter func(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)

// RecommendedVoyage 推荐结果。
type RecommendedVoyage struct {
	LineID          int64   // 航线 ID
	VesselID        int64   // 船舶 ID
	VoyageDate      string  // 航次日期
	VesselName      string  // 船舶名称
	LineName        string  // 航线名称
	MinRemainingCap float64 // 所有航段中的最小剩余容量（瓶颈航段）
}

// VoyageRecommender 航次推荐器。
//
// 推荐策略：
//  1. 遍历每个航次，计算起止港口间所有航段的剩余容量。
//  2. 取所有航段中的最小剩余容量作为该航次的"代表容量"（瓶颈原则）。
//  3. 筛选出最小剩余容量 >= requiredTon 的航次。
//  4. 按最小剩余容量从大到小排序（容量越大的航次排在越前面）。
//
// 瓶颈原则：整条航线的运力受限于容量最小的航段。
// 例：上海→新加坡 剩余 100 吨，新加坡→鹿特丹 剩余 50 吨，
// 则该航次最多只能运 50 吨。取 min(100, 50) = 50。
type VoyageRecommender interface {
	Recommend(voyages []VoyageInfo, startPortID, endPortID int64, requiredTon float64, getRemaining SegmentRemainingGetter) ([]RecommendedVoyage, error)
}

// voyageRecommender 是 VoyageRecommender 接口的私有实现。
type voyageRecommender struct {
	segCalc SegmentCalculator
}

// NewVoyageRecommender 创建航次推荐器实例，依赖航段计算器。
func NewVoyageRecommender(segCalc SegmentCalculator) VoyageRecommender {
	return &voyageRecommender{segCalc: segCalc}
}

// Recommend 执行航次推荐算法。
//
// 步骤：
//  1. 遍历所有航次候选（VoyageInfo）。
//  2. 通过 segCalc.Calculate 计算起止港口间的航段。
//     如果起止港口不在此航线的港口序列中，跳过此航次。
//  3. 对每个航段，调用 getRemaining 获取剩余容量。
//  4. 取所有航段中最小的剩余容量。
//  5. 如果最小剩余容量 >= requiredTon，加入候选列表。
//  6. 按最小剩余容量降序排列候选列表。
//
// 为什么排序是降序而非升序：
//   剩余容量越多的航次越不容易超卖，给货主推荐"最有余裕"的航次。
func (r *voyageRecommender) Recommend(voyages []VoyageInfo, startPortID, endPortID int64, requiredTon float64, getRemaining SegmentRemainingGetter) ([]RecommendedVoyage, error) {
	type candidate struct {
		info     VoyageInfo
		minRem   float64
		segments [][2]int64
	}
	var candidates []candidate

	for _, v := range voyages {
		segs, err := r.segCalc.Calculate(v.PortIDs, startPortID, endPortID)
		if err != nil {
			continue
		}
		var minRem float64 = -1
		ok := true
		for _, seg := range segs {
			rem, err := getRemaining(v.LineID, v.VesselID, v.VoyageDate, seg[0], seg[1])
			if err != nil {
				ok = false
				break
			}
			if minRem == -1 || rem < minRem {
				minRem = rem
			}
			if rem < requiredTon {
				ok = false
				break
			}
		}
		if ok && minRem >= requiredTon {
			candidates = append(candidates, candidate{
				info:     v,
				minRem:   minRem,
				segments: segs,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].minRem > candidates[j].minRem
	})

	result := make([]RecommendedVoyage, len(candidates))
	for i, c := range candidates {
		result[i] = RecommendedVoyage{
			LineID:          c.info.LineID,
			VesselID:        c.info.VesselID,
			VoyageDate:      c.info.VoyageDate,
			VesselName:      c.info.VesselName,
			LineName:        c.info.LineName,
			MinRemainingCap: c.minRem,
		}
	}
	return result, nil
}

