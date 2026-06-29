package idgen

import (
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

var (
	sf   *sonyflake.Sonyflake
	once sync.Once
)

func initSnowflake() {
	once.Do(func() {
		st := sonyflake.Settings{
			StartTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		sf = sonyflake.NewSonyflake(st)
		if sf == nil {
			panic("sonyflake not created")
		}
	})
}

// NextID generates a unique uint64 ID.
func NextID() uint64 {
	initSnowflake()
	id, err := sf.NextID()
	if err != nil {
		// fallback to nanosecond timestamp (not guaranteed unique)
		return uint64(time.Now().UnixNano())
	}
	return id
}
