package engine

import (
	"context"
	"errors"

	"github.com/Elk-123/gotix/engine/internal/storage"
)

var (
	ErrConfigMissing = errors.New("redis client is required")
)

type Engine interface {
	LockSeat(ctx context.Context, eventID string, seatIndex int64, userID string) (bool, error)
	ReleaseSeat(ctx context.Context, eventID string, seatIndex int64) error
}

type engineImpl struct {
	store storage.Provider
	opts  *Options
}

func NewEngine(opts ...Option) (Engine, error) {
	o := defaultOptions()
	for _, fn := range opts {
		fn(o)
	}

	if o.Client == nil {
		return nil, ErrConfigMissing
	}

	return &engineImpl{
		store: storage.NewRedisProvider(o.Client, o.Prefix),
		opts:  o,
	}, nil
}

func (e *engineImpl) LockSeat(ctx context.Context, eventID string, seatIndex int64, userID string) (bool, error) {
	return e.store.Lock(ctx, eventID, seatIndex, userID, e.opts.LockTTL)
}

func (e *engineImpl) ReleaseSeat(ctx context.Context, eventID string, seatIndex int64) error {
	return e.store.Unlock(ctx, eventID, seatIndex)
}