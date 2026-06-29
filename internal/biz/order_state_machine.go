package biz

// OrderStatus constants.
const (
	StatusDraft     int8 = 0
	StatusConfirmed int8 = 1
	StatusInTransit int8 = 2
	StatusCompleted int8 = 3
	StatusCancelled int8 = 4
)

// Allowed transitions map[fromStatus]map[toStatus]bool
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

// OrderStateMachine defines state transition rules.
type OrderStateMachine interface {
	CanTransition(from, to int8) bool
	Transition(from, to int8) error
}

type orderStateMachine struct{}

func NewOrderStateMachine() OrderStateMachine {
	return &orderStateMachine{}
}

func (sm *orderStateMachine) CanTransition(from, to int8) bool {
	if m, ok := allowedTransitions[from]; ok {
		return m[to]
	}
	return false
}

func (sm *orderStateMachine) Transition(from, to int8) error {
	if !sm.CanTransition(from, to) {
		return ErrInvalidStateTransition
	}
	return nil
}
