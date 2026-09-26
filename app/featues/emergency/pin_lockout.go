package emergency

import (
	"sync"
	"time"

	"alert/app/data/repositories"

	"github.com/sirupsen/logrus"
)

// pinLockout counts wrong emergency PINs per branch in Redis. When Redis
// cannot answer it counts in this instance's memory instead, and it always
// uses the higher of the two counts, so a Redis outage never silently
// disables the lockout. A correct PIN is never refused because of an outage:
// emergency alerts must still go out.
type pinLockout struct {
	now func() time.Time

	mu    sync.Mutex
	local map[string]localCount
}

type localCount struct {
	n       int64
	expires time.Time
}

func newPinLockout() *pinLockout {
	return &pinLockout{now: time.Now, local: map[string]localCount{}}
}

// pinLocks is shared by every request this instance serves.
var pinLocks = newPinLockout()

// Attempts returns the wrong-PIN count for key.
func (l *pinLockout) Attempts(store repositories.IRateLimit, key string) int64 {
	remote, err := store.Get(key)
	if err != nil {
		logrus.Warn("pin lockout: Redis unavailable, using in-memory count: ", err)
		remote = 0
	}
	return max(remote, l.localCount(key))
}

// Fail records a wrong PIN and returns the new count.
func (l *pinLockout) Fail(store repositories.IRateLimit, key string, window time.Duration) int64 {
	local := l.localIncrement(key, window)
	remote, err := store.Increment(key, window)
	if err != nil {
		logrus.Warn("pin lockout: Redis unavailable, counting in memory: ", err)
		return local
	}
	return max(remote, local)
}

// Reset clears the count after a correct PIN.
func (l *pinLockout) Reset(store repositories.IRateLimit, key string) {
	l.mu.Lock()
	delete(l.local, key)
	l.mu.Unlock()
	if err := store.Reset(key); err != nil {
		logrus.Warn("pin lockout: could not reset Redis count: ", err)
	}
}

func (l *pinLockout) localCount(key string) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	c, ok := l.local[key]
	if !ok || !l.now().Before(c.expires) {
		delete(l.local, key)
		return 0
	}
	return c.n
}

func (l *pinLockout) localIncrement(key string, window time.Duration) int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	c, ok := l.local[key]
	if !ok || !now.Before(c.expires) {
		c = localCount{expires: now.Add(window)}
	}
	c.n++
	l.local[key] = c
	return c.n
}
