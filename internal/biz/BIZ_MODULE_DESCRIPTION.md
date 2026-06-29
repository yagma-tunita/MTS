================================================================================
               航运管理系统后端 Biz 模块功能说明
               版本：1.0
               生成日期：2026-06-05
================================================================================

本模块位于 internal/biz 目录下，实现了核心业务规则和领域逻辑。所有组件均为
无状态、可复用的纯逻辑单元，不直接依赖 DAO 或数据库连接。它们可以被 service
层调用，用于完成订单创建、运力校验、推荐排序、状态机转换等关键业务操作。

整个 biz 模块的设计遵循“领域驱动设计”思想，每个组件职责单一，便于单元测试。

================================================================================
1. errors.go —— 领域错误定义
================================================================================

定义所有业务逻辑层可能返回的标准错误变量，用于替代简单的字符串错误。

错误列表：
- ErrInvalidPortSequence      : 港口序列 JSON 格式无效
- ErrPortNotFoundInSeq        : 起运港或目的港不在航线港口序列中
- ErrStartAfterEnd            : 起运港出现在目的港之后（顺序错误）
- ErrInsufficientCapacity     : 航段运力不足
- ErrInvalidStateTransition   : 订单状态不允许转换
- ErrInvalidOrderNoFormat     : 订单号格式不正确（保留，未使用）

所有业务方法应优先返回这些预定义错误，上层可通过 errors.Is 判断。

================================================================================
2. port_sequence_parser.go —— 港口序列解析器
================================================================================

将数据库中存储的 JSON 字符串（如 "[1001,1005,1008]"）解析为 []int64 切片。

接口: PortSequenceParser
- Parse(seqJSON string) ([]int64, error)
  输入: 原始 JSON 字符串
  输出: 港口 ID 列表或错误

实现类: portSequenceParser
构造函数: NewPortSequenceParser()

使用场景: service 层获取 shipping_line 后，调用此解析器获得港口顺序列表，
         以便后续计算航段。

================================================================================
3. segment_calculator.go —— 航段计算器
================================================================================

根据完整的港口 ID 序列（按顺序）以及起运港、目的港，计算出所有需要经过的
相邻港口对（航段）。例如：序列 [1001,1005,1008]，起运港=1001，目的港=1008，
则返回 [[1001,1005], [1005,1008]]。

接口: SegmentCalculator
- Calculate(portIDs []int64, startPortID, endPortID int64) ([][2]int64, error)
  输入: 港口 ID 切片，起运港 ID，目的港 ID
  输出: 二维数组，每个元素为 [start, end] 的航段，或错误（端口不在序列或顺序颠倒）

实现类: segmentCalculator
构造函数: NewSegmentCalculator()

使用场景: 订单创建时需要知道订单覆盖了哪些航段，用于后续运力校验和插入占用记录。

================================================================================
4. capacity_checker.go —— 运力校验器
================================================================================

核心业务规则：判断一个新订单（总重量）是否可以在指定航段上被满足。它依赖一个
外部函数来获取每个航段当前已占用的吨位。

接口: CapacityChecker
- Check(segments [][2]int64, maxWeight float64, 
        occupiedGetter func([2]int64) (float64, error), 
        totalWeight float64) (bool, float64, error)
  参数:
    - segments        : 航段列表（由 SegmentCalculator 计算得到）
    - maxWeight       : 船舶最大载重吨位
    - occupiedGetter  : 回调函数，输入一个航段，返回该航段已占用的吨位
    - totalWeight     : 本次订单总重量
  返回:
    - bool            : 是否所有航段都能容纳
    - float64         : 所有航段中的最小剩余容量（用于推荐排序）
    - error           : 若获取占用时出错则返回

实现类: capacityChecker
构造函数: NewCapacityChecker()

使用场景: 在订单创建时，遍历所有航段，调用 DAO 获取已占吨位，然后判断是否超限。
         返回的最小剩余容量可用于前端展示。

================================================================================
5. order_no_generator.go —— 订单号生成器
================================================================================

生成全局唯一的订单号，格式为：前缀 + YYYYMMDD + 8位随机十六进制字符。
默认前缀为 "ORD"。

接口: OrderNoGenerator
- Generate() string
  无输入，返回新生成的订单号（例如 "ORD20250605a3f2b1c8"）

实现类: orderNoGenerator
构造函数: NewOrderNoGenerator(prefix string) 
          传入空字符串时使用默认前缀 "ORD"。

使用场景: 创建订单时生成 order_no 字段值。可扩展为支持从数据库序列或 Redis
         获取序号，但当前纯随机方案已满足唯一性要求（碰撞概率极低）。

================================================================================
6. cost_calculator.go —— 费用计算器
================================================================================

根据货物明细列表计算总重量、总体积、总费用以及每项货物的小计。这是一个纯业务
规则，确保后端不依赖前端传入的总金额（防止篡改）。

接口: CostCalculator
- Calculate(items []CargoItem) (*CostResult, error)
  输入: 货物列表，每个货物包含重量、体积、单价、数量
  输出: CostResult 结构体，包含:
        - TotalWeightTon   : 总重量
        - TotalVolumeM3    : 总体积
        - TotalCost        : 总费用
        - ItemsSubtotal    : 每项货物小计（切片）

数据结构: CargoItem
- WeightTon   float64
- VolumeM3    float64
- UnitPrice   float64
- Quantity    float64

实现类: costCalculator
构造函数: NewCostCalculator()

使用场景: service 层接收前端货物列表后，调用此计算器得到 totals，存入订单表中。
         如果前端传入了 totals，后端仍应重新计算并覆盖，确保数据一致性。

================================================================================
7. order_state_machine.go —— 订单状态机
================================================================================

定义订单允许的状态转换规则，防止非法状态变更（如从“已完成”改回“运输中”）。

状态常量:
- StatusDraft     = 0   // 草稿
- StatusConfirmed = 1   // 已确认
- StatusInTransit = 2   // 运输中
- StatusCompleted = 3   // 已完成
- StatusCancelled = 4   // 已取消

允许的转换（map[from]map[to]bool）:
  草稿 → 已确认, 取消
  已确认 → 运输中, 取消
  运输中 → 已完成, 取消
  已完成 → 无
  取消 → 无

接口: OrderStateMachine
- CanTransition(from, to int8) bool      // 判断是否允许转换
- Transition(from, to int8) error        // 执行转换，若不允许返回 ErrInvalidStateTransition

实现类: orderStateMachine
构造函数: NewOrderStateMachine()

使用场景: 在 service 层更新订单状态前，调用此状态机校验合法性。例如：
          if err := biz.OrderStateMachine.Transition(oldStatus, newStatus); err != nil {
              return err
          }

================================================================================
8. voyage_recommender.go —— 推荐排序算法（核心业务）
================================================================================

根据用户提供的起运港、目的港、需求吨位，对一组候选航次进行过滤和排序，返回
按剩余容量降序（最宽松优先）的推荐列表。

输入要求：
- 每个候选航次需提供：航线 ID、船次 ID、航次日期、船名、航线名、船舶最大载重、
  以及该航线的港口 ID 序列（已解析）。
- 外部还需提供一个函数 getRemaining，用于实时获取指定航段剩余容量（由 service 层
  通过 DAO 实现）。

接口: VoyageRecommender
- Recommend(voyages []VoyageInfo, startPortID, endPortID int64, requiredTon float64, 
            getRemaining SegmentRemainingGetter) ([]RecommendedVoyage, error)

参数说明:
  - voyages       : 候选航次信息切片
  - startPortID   : 起运港 ID
  - endPortID     : 目的港 ID
  - requiredTon   : 订单需求吨位
  - getRemaining  : 回调函数，签名 (lineID, vesselID, voyageDate, startPort, endPort) (float64, error)
                    service 层应将其与 DAO 方法绑定。

返回:
  - RecommendedVoyage 切片，按 MinRemainingCap 降序排列。

算法步骤:
  1. 对每个候选航次，使用 SegmentCalculator 计算其从 startPort 到 endPort 的所有航段。
  2. 对每个航段，调用 getRemaining 获取当前剩余容量。
  3. 取所有航段中的最小值作为该航次的“瓶颈剩余容量”。
  4. 如果瓶颈剩余容量 >= requiredTon，则该航次可用。
  5. 所有可用航次按瓶颈剩余容量降序排序（容量最大的排前面）。

实现类: voyageRecommender
构造函数: NewVoyageRecommender(segCalc SegmentCalculator)

使用场景: 用户通过前端输入起运/目的港和货物重量，请求推荐航次。service 层从数据库
         查询所有可能的航线、航次，组装 VoyageInfo 列表，然后调用此推荐器进行过滤
         和排序，最后返回给前端供用户选择。

================================================================================
9. container.go —— 聚合容器（可选）
================================================================================

为了方便 service 层统一获取所有业务组件，提供了一个简单的容器结构体。

结构体: BizContainer
字段:
  - PortSequenceParser
  - SegmentCalculator
  - CapacityChecker
  - OrderNoGenerator
  - CostCalculator
  - OrderStateMachine
  - VoyageRecommender

构造函数: NewBizContainer() *BizContainer
  内部创建 SegmentCalculator 并共享给 VoyageRecommender。

使用示例（在 service 初始化时）:
  bizContainer := biz.NewBizContainer()
  // 然后分别注入到各个 service 中，或直接使用 bizContainer.SegmentCalculator 等。

================================================================================
10. 模块间的协作示例
================================================================================

以下是一个典型的订单创建流程中，biz 各组件如何协作：

1. service 层接收请求后，调用 biz.PortSequenceParser.Parse() 解析航线港口序列。
2. 调用 biz.SegmentCalculator.Calculate() 计算订单覆盖的航段列表。
3. 调用 biz.CostCalculator.Calculate() 计算总重量和总费用。
4. 定义 occupiedGetter 回调，内部调用 DAO 查询已占吨位。
5. 调用 biz.CapacityChecker.Check() 校验运力是否充足。
6. 调用 biz.OrderNoGenerator.Generate() 生成订单号。
7. 如果涉及状态变更，调用 biz.OrderStateMachine.Transition() 校验合法性。
8. 所有校验通过后，执行数据库事务。

推荐排序场景：
- service 层查询所有候选航次，组装成 []VoyageInfo。
- 定义 getRemaining 回调（内部调用 DAO）。
- 调用 biz.VoyageRecommender.Recommend() 获得排序后的推荐列表。

================================================================================
11. 单元测试优势
================================================================================

biz 层所有组件都是纯逻辑，不依赖数据库，因此可以轻松编写单元测试：

示例（对 CapacityChecker 的测试）：
  checker := NewCapacityChecker()
  segments := [][2]int64{{1,2},{2,3}}
  maxWeight := 100.0
  occupiedGetter := func(seg [2]int64) (float64, error) {
      // 模拟已占吨位
      if seg[0]==1 && seg[1]==2 { return 30, nil }
      return 10, nil
  }
  ok, minRem, err := checker.Check(segments, maxWeight, occupiedGetter, 50)
  // 预期 ok = true (因为 100-30-50=20 >=0, 100-10-50=40 >=0)
  // 预期 minRem = 20 (两个航段中较小的剩余)

这允许快速验证业务规则，无需启动数据库或 mock 复杂环境。

================================================================================
                              文档结束
================================================================================