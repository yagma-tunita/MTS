package biz

import (
	"encoding/json"
)

// PortSequenceParser 港口序列解析器。
//
// 港口序列在数据库中存储为 JSON 数组字符串，例如 "[1, 2, 3, 5, 7]"。
// 这个解析器将其转换为 []int64，供 SegmentCalculator 和 VoyageRecommender 使用。
//
// 为什么使用 JSON 存储而不是关联表：
//   每条航线的港口数通常只有 3-8 个，为这么少的数据单独建一张关联表
//   会导致每次查询多一次 JOIN（line_port → port），而我们对港口序列的
//   读取频率非常高（每次下单都需要解析）。JSON 一次读取即可。
type PortSequenceParser interface {
	Parse(seqJSON string) ([]int64, error)
}

// portSequenceParser 是 PortSequenceParser 接口的私有实现。
type portSequenceParser struct{}

// NewPortSequenceParser 创建港口序列解析器实例。
func NewPortSequenceParser() PortSequenceParser {
	return &portSequenceParser{}
}

// Parse 将 JSON 数组字符串解析为港口 ID 切片。
//
// 输入示例："[1, 2, 3]"
// 输出示例：[1, 2, 3]
//
// 如果 JSON 解析失败（如格式不正确），返回 error。
// 注意：空数组 "[]" 是合法的，返回空切片。
func (p *portSequenceParser) Parse(seqJSON string) ([]int64, error) {
	var ids []int64
	if err := json.Unmarshal([]byte(seqJSON), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}

