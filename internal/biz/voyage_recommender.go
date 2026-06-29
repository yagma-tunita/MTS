package biz

// VoyageInfo contains minimal data for a voyage candidate.
type VoyageInfo struct {
	LineID     int64
	VesselID   int64
	VoyageDate string
	VesselName string
	LineName   string
	MaxWeight  float64
	PortIDs    []int64
}

// SegmentRemainingGetter is a function that returns remaining capacity for a segment.
type SegmentRemainingGetter func(lineID, vesselID int64, voyageDate string, startPortID, endPortID int64) (float64, error)

// RecommendedVoyage represents a sorted recommendation.
type RecommendedVoyage struct {
	LineID          int64
	VesselID        int64
	VoyageDate      string
	VesselName      string
	LineName        string
	MinRemainingCap float64
}

// VoyageRecommender provides sorting and filtering of voyages by capacity.
type VoyageRecommender interface {
	// Recommend filters voyages that can accommodate requiredTon and sorts by remaining capacity descending.
	Recommend(voyages []VoyageInfo, startPortID, endPortID int64, requiredTon float64, getRemaining SegmentRemainingGetter) ([]RecommendedVoyage, error)
}

type voyageRecommender struct {
	segCalc SegmentCalculator
}

func NewVoyageRecommender(segCalc SegmentCalculator) VoyageRecommender {
	return &voyageRecommender{segCalc: segCalc}
}

func (r *voyageRecommender) Recommend(voyages []VoyageInfo, startPortID, endPortID int64, requiredTon float64, getRemaining SegmentRemainingGetter) ([]RecommendedVoyage, error) {
	type candidate struct {
		info     VoyageInfo
		minRem   float64
		segments [][2]int64
	}
	var candidates []candidate

	for _, v := range voyages {
		// compute segments for this voyage
		segs, err := r.segCalc.Calculate(v.PortIDs, startPortID, endPortID)
		if err != nil {
			continue // skip voyages where ports not in sequence
		}
		// compute min remaining capacity
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

	// sort by minRem descending (best first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[i].minRem < candidates[j].minRem {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

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
