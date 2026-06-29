package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"backend/pkg/timeutil"

	"gorm.io/gorm"
)

var Logger = slog.Default()

// ServiceError is deprecated, use pkg/errors instead.
type ServiceError struct {
	Code    string
	Message string
}

func (e ServiceError) Error() string {
	return e.Message
}

// Deprecated error codes, for backward compatibility only.
const (
	ErrCodeOrderNotFound   = "ORDER_NOT_FOUND"
	ErrCodeInsufficientCap = "INSUFFICIENT_CAPACITY"
	ErrCodeLockFailed      = "LOCK_FAILED"
	ErrCodeInvalidPortSeq  = "INVALID_PORT_SEQUENCE"
	ErrCodeNoCargoNote     = "CARGO_NOTE_NOT_FOUND"
	ErrCodeVesselNotFound  = "VESSEL_NOT_FOUND"
	ErrCodeLineNotFound    = "LINE_NOT_FOUND"
)

func AcquireLock(tx *gorm.DB, lockName string, timeoutSec int) (bool, error) {
	var result int
	err := tx.Raw("SELECT GET_LOCK(?, ?)", lockName, timeoutSec).Scan(&result).Error
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

func ReleaseLock(tx *gorm.DB, lockName string) error {
	var result int
	return tx.Raw("SELECT RELEASE_LOCK(?)", lockName).Scan(&result).Error
}

func VoyageLockKey(lineID, vesselID int64, voyageDate string) string {
	return fmt.Sprintf("voyage_%d_%d_%s", lineID, vesselID, voyageDate)
}

// MustParseDate converts YYYY-MM-DD to time.Time using timeutil.
func MustParseDate(s string) time.Time {
	t, _ := timeutil.ParseDate(s)
	return t
}

func PtrInt8(v int8) *int8 {
	return &v
}

func WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 30*time.Second)
}
