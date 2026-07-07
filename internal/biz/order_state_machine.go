package biz

// 订单状态常量。
//
// 状态定义：
//   0 - Pending（待确认）：货主已提交，等待海运公司审核。
//   1 - Confirmed（已确认）：海运公司已确认，开始安排运输。
//   2 - In Transit（运输中）：货物已装船，正在运输途中。
//   3 - Completed（已完成）：货物已送达，订单完成。
//   4 - Cancelled（已取消）：订单被取消。
const (
	StatusDraft     int8 = 0
	StatusConfirmed int8 = 1
	StatusInTransit int8 = 2
	StatusCompleted int8 = 3
	StatusCancelled int8 = 4
)

// allowedTransitions 定义订单状态的合法转换规则。
//
// 转换图：
//   0(Draft) ──→ 1(Confirmed)
//   0(Draft) ──→ 4(Cancelled)
//   1(Confirmed) ──→ 2(In Transit)
//   1(Confirmed) ──→ 4(Cancelled)
//   2(In Transit) ──→ 3(Completed)
//   2(In Transit) ──→ 4(Cancelled)
//   3(Completed) ──→ 无（终态）
//   4(Cancelled) ──→ 无（终态）
//
// 设计原则：从任何状态都可以取消，但一旦完成或取消就不能再改变。
var allowedTransitions = map[int8]map[int8]bool{
	StatusDraft: {
		StatusConfirmed: true,
		StatusCancelled: true,
	},
	StatusConfirmed: {
		StatusInTransit: true,
		StatusCancelled: true,
	},
	StatusInTransit: {
		StatusCompleted: true,
		StatusCancelled: true,
	},
	StatusCompleted: {},
	StatusCancelled: {},
}

// OrderStateMachine 订单状态机，校验状态变更是否合法。
//
// 使用 map 实现，而非 switch-case 或状态模式：
//   map 的配置式和数据驱动风格。如果要修改状态转换规则，
//   只需要改 allowedTransitions 这个数据，不需要改代码逻辑。
type OrderStateMachine interface {
	CanTransition(from, to int8) bool
	Transition(from, to int8) error
}

// orderStateMachine 是 OrderStateMachine 接口的私有实现。
type orderStateMachine struct{}

// NewOrderStateMachine 创建订单状态机实例。
func NewOrderStateMachine() OrderStateMachine {
	return &orderStateMachine{}
}

// CanTransition 检查 from → to 的转换是否在 allowedTransitions 中定义。
func (sm *orderStateMachine) CanTransition(from, to int8) bool {
	if m, ok := allowedTransitions[from]; ok {
		return m[to]
	}
	return false
}

// Transition 执行状态转换校验，非法转换返回 ErrInvalidStateTransition。
func (sm *orderStateMachine) Transition(from, to int8) error {
	if !sm.CanTransition(from, to) {
		return ErrInvalidStateTransition
	}
	return nil
}

