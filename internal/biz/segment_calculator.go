package biz

// SegmentCalculator computes adjacent port pairs between start and end.
type SegmentCalculator interface {
	Calculate(portIDs []int64, startPortID, endPortID int64) ([][2]int64, error)
}

type segmentCalculator struct{}

func NewSegmentCalculator() SegmentCalculator {
	return &segmentCalculator{}
}

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
