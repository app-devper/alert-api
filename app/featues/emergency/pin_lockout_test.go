package emergency

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

var errRedisDown = errors.New("redis: connection refused")

// fakeRateLimit mimics Redis counters; down makes every call fail.
type fakeRateLimit struct {
	counts map[string]int64
	down   bool
}

func newFakeRateLimit() *fakeRateLimit { return &fakeRateLimit{counts: map[string]int64{}} }

func (f *fakeRateLimit) Increment(key string, _ time.Duration) (int64, error) {
	if f.down {
		return 0, errRedisDown
	}
	f.counts[key]++
	return f.counts[key], nil
}
func (f *fakeRateLimit) Get(key string) (int64, error) {
	if f.down {
		return 0, errRedisDown
	}
	return f.counts[key], nil
}
func (f *fakeRateLimit) Reset(key string) error {
	if f.down {
		return errRedisDown
	}
	delete(f.counts, key)
	return nil
}

type fakeAuditLog struct {
	repositories.IAuditLog
	records []entities.AuditLog
}

func (f *fakeAuditLog) Record(log entities.AuditLog) { f.records = append(f.records, log) }

type testClock struct{ t time.Time }

func newTestLockout() (*pinLockout, *testClock) {
	clk := &testClock{t: time.Date(2026, 9, 27, 9, 0, 0, 0, time.UTC)}
	l := newPinLockout()
	l.now = func() time.Time { return clk.t }
	return l, clk
}

const window = time.Duration(constant.PinLockMinutes) * time.Minute

func TestLockoutCountsInRedisWhenHealthy(t *testing.T) {
	l, _ := newTestLockout()
	store := newFakeRateLimit()
	for i := 0; i < 3; i++ {
		l.Fail(store, "k", window)
	}
	if got := l.Attempts(store, "k"); got != 3 || store.counts["k"] != 3 {
		t.Fatalf("expected 3 attempts in Redis, got %d (redis %d)", got, store.counts["k"])
	}
	l.Reset(store, "k")
	if got := l.Attempts(store, "k"); got != 0 {
		t.Fatalf("expected reset to clear the count, got %d", got)
	}
}

func TestLockoutStillCountsWhenRedisIsDown(t *testing.T) {
	l, _ := newTestLockout()
	store := newFakeRateLimit()
	store.down = true
	var last int64
	for i := 0; i < constant.PinMaxAttempts; i++ {
		last = l.Fail(store, "k", window)
	}
	if last != constant.PinMaxAttempts || l.Attempts(store, "k") < constant.PinMaxAttempts {
		t.Fatalf("expected the in-memory count to reach the lock limit, got fail=%d attempts=%d", last, l.Attempts(store, "k"))
	}
}

func TestOutageCountsSurviveRedisRecovery(t *testing.T) {
	l, _ := newTestLockout()
	store := newFakeRateLimit()
	store.down = true
	for i := 0; i < constant.PinMaxAttempts; i++ {
		l.Fail(store, "k", window)
	}
	store.down = false
	if got := l.Attempts(store, "k"); got < constant.PinMaxAttempts {
		t.Fatalf("attempts during the outage must still lock after recovery, got %d", got)
	}
}

func TestInMemoryCountExpiresWithTheWindow(t *testing.T) {
	l, clk := newTestLockout()
	store := newFakeRateLimit()
	store.down = true
	l.Fail(store, "k", window)
	clk.t = clk.t.Add(window)
	if got := l.Attempts(store, "k"); got != 0 {
		t.Fatalf("expected the in-memory count to expire, got %d", got)
	}
}

func verify(t *testing.T, repo *domain.Repository, setting entities.BranchSetting, pin string) (bool, int) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	ok := verifyPin(ctx, repo, setting, pin)
	return ok, w.Code
}

func TestVerifyPinDuringRedisOutage(t *testing.T) {
	pinLocks, _ = newTestLockout()
	t.Cleanup(func() { pinLocks = newPinLockout() })

	hash, _ := bcrypt.GenerateFromPassword([]byte("1234"), bcrypt.MinCost)
	setting := entities.BranchSetting{ClientId: "c1", BranchId: "b1", PinHash: string(hash)}
	store := newFakeRateLimit()
	store.down = true
	audit := &fakeAuditLog{}
	repo := &domain.Repository{RateLimit: store, AuditLog: audit}

	if ok, code := verify(t, repo, setting, "1234"); !ok {
		t.Fatalf("a correct PIN must still send the alert while Redis is down, got %d", code)
	}
	for i := 0; i < constant.PinMaxAttempts; i++ {
		if ok, code := verify(t, repo, setting, "0000"); ok || code != http.StatusForbidden {
			t.Fatalf("wrong PIN %d: expected 403, got ok=%v code=%d", i+1, ok, code)
		}
	}
	if ok, code := verify(t, repo, setting, "1234"); ok || code != http.StatusLocked {
		t.Fatalf("after %d wrong PINs the branch must be locked even for the correct PIN, got ok=%v code=%d", constant.PinMaxAttempts, ok, code)
	}
	if len(audit.records) != 1 {
		t.Fatalf("expected one PIN_LOCKED audit record, got %d", len(audit.records))
	}
}
