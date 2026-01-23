package engine

import (
	"time"

	"github.com/redis/go-redis/v9"
)

type Options struct {
	Client  *redis.Client
	LockTTL time.Duration
	Prefix  string
}

type Option func(*Options)

func defaultOptions() *Options {
	return &Options{
		LockTTL: 5 * time.Minute,
		Prefix:  "gotix",
	}
}

func WithRedis(rdb *redis.Client) Option {
	return func(o *Options) {
		o.Client = rdb
	}
}

func WithLockTTL(d time.Duration) Option {
	return func(o *Options) {
		o.LockTTL = d
	}
}

func WithPrefix(p string) Option {
	return func(o *Options) {
		o.Prefix = p
	}
}