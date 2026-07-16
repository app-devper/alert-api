package alerting

import (
	"strings"
	"testing"
)

func TestValidateNoLinkRejectsHttpUrl(t *testing.T) {
	if err := ValidateNoLink("อพยพด่วน ดูรายละเอียด http://evil.example"); err == nil {
		t.Fatal("expected link rejection")
	}
}

func TestValidateNoLinkRejectsBareDomain(t *testing.T) {
	for _, text := range []string{"visit example.com now", "go to www.shop.th", "see shop.app/deals"} {
		if err := ValidateNoLink(text); err == nil {
			t.Fatalf("expected rejection for %q", text)
		}
	}
}

func TestValidateNoLinkAcceptsPlainEmergencyText(t *testing.T) {
	text := "แจ้งเหตุฉุกเฉิน: พบเหตุเพลิงไหม้ กรุณาออกจากร้านทันที ห้ามใช้ลิฟต์"

	if err := ValidateNoLink(text); err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
}

func TestValidateNoLinkAcceptsEnglishEmergencyText(t *testing.T) {
	text := "FIRE ALERT: Fire reported. Exit immediately via the nearest emergency exit. Do not use elevators."

	if err := ValidateNoLink(text); err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
}

func TestSmsSegmentCountAsciiSingleSegment(t *testing.T) {
	if count := SmsSegmentCount(strings.Repeat("a", 160)); count != 1 {
		t.Fatalf("expected 1 segment, got %d", count)
	}
}

func TestSmsSegmentCountAsciiMultiSegment(t *testing.T) {
	if count := SmsSegmentCount(strings.Repeat("a", 161)); count != 2 {
		t.Fatalf("expected 2 segments, got %d", count)
	}
}

func TestSmsSegmentCountThaiUsesUcsLimits(t *testing.T) {
	if count := SmsSegmentCount(strings.Repeat("ก", 70)); count != 1 {
		t.Fatalf("expected 1 segment, got %d", count)
	}
	if count := SmsSegmentCount(strings.Repeat("ก", 71)); count != 2 {
		t.Fatalf("expected 2 segments, got %d", count)
	}
}

func TestSmsSegmentCountEmptyText(t *testing.T) {
	if count := SmsSegmentCount(""); count != 0 {
		t.Fatalf("expected 0 segments, got %d", count)
	}
}
