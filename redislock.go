package redlockimp

import "context"

type Locker interface {
	Lock(ctx context.Context) error
	Unlock(ctx context.Context) error
}
