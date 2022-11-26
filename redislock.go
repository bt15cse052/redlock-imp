package redlockimp

type Locker interface {
	Lock(key string) error
	Unlock() error
}
