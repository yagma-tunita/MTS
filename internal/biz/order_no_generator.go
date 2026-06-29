package biz

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// OrderNoGenerator creates unique order numbers.
type OrderNoGenerator interface {
	Generate() string
}

type orderNoGenerator struct {
	prefix string
}

func NewOrderNoGenerator(prefix string) OrderNoGenerator {
	if prefix == "" {
		prefix = "ORD"
	}
	return &orderNoGenerator{prefix: prefix}
}

func (g *orderNoGenerator) Generate() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s%s%s", g.prefix, time.Now().Format("20060102"), hex.EncodeToString(b))
}
