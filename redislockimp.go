package redlockimp

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
)

type locker struct {
	Clients []*redis.Client
	KeyName string
	KeyVal  string
	Quorum  int32
	Expiry  time.Duration
	Drift   time.Duration
}

func (l *locker) Lock(ctx context.Context) error {
	var scriptLock = `
	if redis.call("EXISTS", KEYS[1]) == 1 then
		return 0
	end
	return redis.call("SET", KEYS[1], ARGV[1], "PX", ARGV[2])
	`
	totallocks := 0
	for _, rc := range l.Clients {
		start := time.Now()
		res, err := rc.Eval(ctx, scriptLock, []string{l.KeyName}, l.KeyVal, l.Expiry.Milliseconds()).Result()
		if err != nil {
			return err
		}
		ok := res == "OK"
		now := time.Now()
		if ok && (l.Expiry-now.Sub(start)-l.Drift) > 0 {
			totallocks++
		}
	}
	if totallocks < int(l.Quorum) {

		return fmt.Errorf("unable to accquire lock")
	}

	return nil
}

func (l *locker) Unlock(ctx context.Context) error {
	totalSuccess := 0
	var scriptUnlock = `
	if redis.call("GET", KEYS[1]) == ARGV[1] then
		return redis.call("DEL", KEYS[1])
	else
		return 0
	end
	`
	for _, rc := range l.Clients {

		status, err := rc.Eval(nil, scriptUnlock, []string{l.KeyName}, l.KeyVal).Result()
		if err != nil {
			return err
		}

		if status != int64(0) {
			totalSuccess++
		}
	}

	if totalSuccess < int(l.Quorum) {
		return fmt.Errorf("unable to release lock")
	}

	return nil
}

func GetNewLocker(key string, clients []*redis.Client, ttl time.Duration, quorum int, drift time.Duration) (*locker, error) {
	return &locker{
		Clients: clients,
		KeyName: key,
		KeyVal:  generateRandomString(),
		Quorum:  int32(quorum),
		Expiry:  ttl,
		Drift:   drift,
	}, nil
}
func generateRandomString() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune,
		time.Now().Unix()%64)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
