package biz

import (
	"encoding/json"
)

// PortSequenceParser defines port sequence parsing.
type PortSequenceParser interface {
	Parse(seqJSON string) ([]int64, error)
}

type portSequenceParser struct{}

func NewPortSequenceParser() PortSequenceParser {
	return &portSequenceParser{}
}

func (p *portSequenceParser) Parse(seqJSON string) ([]int64, error) {
	var ids []int64
	if err := json.Unmarshal([]byte(seqJSON), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
