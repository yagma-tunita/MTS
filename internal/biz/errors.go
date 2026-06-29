package biz

import "errors"

var (
	ErrInvalidPortSequence    = errors.New("invalid port sequence")
	ErrPortNotFoundInSeq      = errors.New("start or end port not found in sequence")
	ErrStartAfterEnd          = errors.New("start port appears after end port")
	ErrInsufficientCapacity   = errors.New("insufficient capacity on segment")
	ErrInvalidStateTransition = errors.New("invalid order state transition")
	ErrInvalidOrderNoFormat   = errors.New("invalid order number format")
	ErrEmptyCargoList         = errors.New("cargo list cannot be empty")
)
