package alerting

import (
	"testing"
	"time"

	"alert/app/core/constant"
)

func TestCooldownRemainingZeroWhenNoPriorEvent(t *testing.T) {
	if remaining := CooldownRemaining(nil, time.Now(), 180); remaining != 0 {
		t.Fatalf("expected 0, got %v", remaining)
	}
}

func TestCooldownRemainingBlocksWithinWindow(t *testing.T) {
	now := time.Now()
	lastSent := now.Add(-60 * time.Second)

	remaining := CooldownRemaining(&lastSent, now, 180)

	if remaining != 120*time.Second {
		t.Fatalf("expected 120s remaining, got %v", remaining)
	}
}

func TestCooldownRemainingZeroAfterWindow(t *testing.T) {
	now := time.Now()
	lastSent := now.Add(-181 * time.Second)

	if remaining := CooldownRemaining(&lastSent, now, 180); remaining != 0 {
		t.Fatalf("expected 0, got %v", remaining)
	}
}

func TestNormalizeCooldownSecondsClampsToRange(t *testing.T) {
	if normalized := NormalizeCooldownSeconds(0); normalized != constant.DefaultCooldownSeconds {
		t.Fatalf("expected default %d, got %d", constant.DefaultCooldownSeconds, normalized)
	}
	if normalized := NormalizeCooldownSeconds(600); normalized != constant.MaxCooldownSeconds {
		t.Fatalf("expected max %d, got %d", constant.MaxCooldownSeconds, normalized)
	}
	if normalized := NormalizeCooldownSeconds(90); normalized != 90 {
		t.Fatalf("expected 90, got %d", normalized)
	}
}

func TestAllClearAndTestAreCooldownExempt(t *testing.T) {
	if !IsCooldownExempt(constant.EventAllClear) {
		t.Fatal("ALL_CLEAR must be cooldown exempt")
	}
	if !IsCooldownExempt(constant.EventTest) {
		t.Fatal("TEST must be cooldown exempt")
	}
	if IsCooldownExempt(constant.EventFire) {
		t.Fatal("FIRE must respect cooldown")
	}
}
