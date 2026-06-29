package biz

// CapacityChecker validates if a new order fits into vessel's capacity on all segments.
type CapacityChecker interface {
	// Check returns true if totalWeight fits on all segments.
	// occupiedGetter is a function that returns already occupied tons for a given segment.
	Check(segments [][2]int64, maxWeight float64, occupiedGetter func(seg [2]int64) (float64, error), totalWeight float64) (bool, float64, error)
}

type capacityChecker struct{}

func NewCapacityChecker() CapacityChecker {
	return &capacityChecker{}
}

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
