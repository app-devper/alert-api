package alerting

import (
	"regexp"
	"testing"
	"time"
)

func TestGenerateOtpIsSixDigits(t *testing.T) {
	pattern := regexp.MustCompile(`^\d{6}$`)
	for i := 0; i < 20; i++ {
		otp, err := GenerateOtp()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !pattern.MatchString(otp) {
			t.Fatalf("expected 6 digits, got %s", otp)
		}
	}
}

func TestGenerateRefCodeIsFourSafeChars(t *testing.T) {
	pattern := regexp.MustCompile(`^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{4}$`)
	for i := 0; i < 20; i++ {
		refCode, err := GenerateRefCode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !pattern.MatchString(refCode) {
			t.Fatalf("expected 4 safe chars, got %s", refCode)
		}
	}
}

func TestHashOtpIsDeterministicAndSecretBound(t *testing.T) {
	first := HashOtp("secret", "+66812345678", "AB3K", "123456")
	second := HashOtp("secret", "+66812345678", "AB3K", "123456")
	differentSecret := HashOtp("other", "+66812345678", "AB3K", "123456")
	differentOtp := HashOtp("secret", "+66812345678", "AB3K", "654321")

	if first != second {
		t.Fatal("hash must be deterministic")
	}
	if first == differentSecret {
		t.Fatal("hash must depend on secret")
	}
	if first == differentOtp {
		t.Fatal("hash must depend on otp")
	}
}

func TestGenerateSessionTokenIsUnique(t *testing.T) {
	first, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first == second {
		t.Fatal("tokens must be unique")
	}
	if len(first) != 64 {
		t.Fatalf("expected 64 hex chars, got %d", len(first))
	}
}

func TestFormatEventNoMatchesSpec(t *testing.T) {
	date := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	if eventNo := FormatEventNo("EM", date, 1); eventNo != "EM260716001" {
		t.Fatalf("expected EM260716001, got %s", eventNo)
	}
	if eventNo := FormatEventNo("TS", date, 12); eventNo != "TS260716012" {
		t.Fatalf("expected TS260716012, got %s", eventNo)
	}
}

func TestFormatCheckInNoMatchesSpec(t *testing.T) {
	date := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)

	if checkInNo := FormatCheckInNo(date, 1); checkInNo != "CI2607160001" {
		t.Fatalf("expected CI2607160001, got %s", checkInNo)
	}
}
