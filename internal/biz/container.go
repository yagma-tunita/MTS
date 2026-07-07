package biz

// BizContainer 是 biz 层的"组件箱"，聚合所有纯业务逻辑组件。
//
// 每个字段都是一个业务组件接口。main 函数通过 NewBizContainer()
// 一次性创建所有组件，然后按需注入到各个 Service 的构造函数中。
//
// 组件列表及功能速查：
//
//   PortSequenceParser:
//     解析 JSON 格式的港口序列。输入: "[1,2,3]"，输出: [1,2,3]。
//     用于在创建订单和推荐航次时解析航线的港口顺序。
//
//   SegmentCalculator:
//     航段计算器。给定港口序列和起止港口，计算途径的所有邻接航段。
//     例如港口序列 [1,2,3,5]，起点=1，终点=5，输出 [(1,2),(2,3),(3,5)]。
//     用于将订单拆分为航段，以便按航段管理和校验运力。
//
//   CapacityChecker:
//     容量校验器。遍历所有航段，检查每个航段上
//     "已占用的吨位 + 新订单吨位 <= 船舶最大载重"。
//     如果任何一个航段超容，返回 ErrInsufficientCapacity。
//     这是防止运力超卖的核心校验。
//
//   CostCalculator:
//     成本计算器。汇总多个货物条目的总重量、总体积、总金额。
//     注意：它只做求和，不做运费计算。
//     完整的运费 = 总重量 × 总航程 × 基础费率 × 货物系数
//     在 service 层完成（涉及 config 中的费率配置）。
//
//   OrderNoGenerator:
//     订单号生成器。格式：ORD20260706a3f2c1b0
//     = 前缀(ORD) + 日期(YYYMMDD) + 8位随机hex(基于crypto/rand)
//
//   OrderStateMachine:
//     订单状态机。定义并校验状态转换规则：
//       0(草稿) → 1(已确认)、0(草稿) → 4(已取消)
//       1(已确认) → 2(运输中)、1(已确认) → 4(已取消)
//       2(运输中) → 3(已完成)、2(运输中) → 4(已取消)
//       3(已完成)/4(已取消) → 终态，不可转换
//
//   VoyageRecommender:
//     航次推荐器。推荐算法：遍历所有航次 → 计算每个航次的起止港航段 →
//     取瓶颈段（最小剩余容量）→ 筛选 >= 需求吨位 → 按容量降序排列。
type BizContainer struct {
	PortSequenceParser PortSequenceParser
	SegmentCalculator  SegmentCalculator
	CapacityChecker    CapacityChecker
	OrderNoGenerator   OrderNoGenerator
	CostCalculator     CostCalculator
	OrderStateMachine  OrderStateMachine
	VoyageRecommender  VoyageRecommender
}

// NewBizContainer 创建并初始化所有业务组件实例。
//
// 注意：VoyageRecommender 内部依赖 SegmentCalculator，
// 所以先创建 segCalc 再传给 NewVoyageRecommender。
// 其他组件相互独立，创建顺序无要求。
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
