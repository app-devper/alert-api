package alerting

import "testing"

func TestNormalizeThaiPhoneAcceptsLocalFormat(t *testing.T) {
	normalized, err := NormalizeThaiPhone("0812345678")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "+66812345678" {
		t.Fatalf("expected +66812345678, got %s", normalized)
	}
}

func TestNormalizeThaiPhoneAcceptsInternationalFormat(t *testing.T) {
	normalized, err := NormalizeThaiPhone("+66912345678")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "+66912345678" {
		t.Fatalf("expected +66912345678, got %s", normalized)
	}
}

func TestNormalizeThaiPhoneAcceptsDashesAndSpaces(t *testing.T) {
	normalized, err := NormalizeThaiPhone("081-234-5678")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if normalized != "+66812345678" {
		t.Fatalf("expected +66812345678, got %s", normalized)
	}
}

func TestNormalizeThaiPhoneRejectsLandline(t *testing.T) {
	if _, err := NormalizeThaiPhone("021234567"); err == nil {
		t.Fatal("expected error for landline number")
	}
}

func TestNormalizeThaiPhoneRejectsGarbage(t *testing.T) {
	for _, input := range []string{"", "abc", "12345", "+15551234567"} {
		if _, err := NormalizeThaiPhone(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestMaskPhoneHidesMiddleDigits(t *testing.T) {
	masked := MaskPhone("+66812345678")

	if masked != "+6681XXX5678" {
		t.Fatalf("expected +6681XXX5678, got %s", masked)
	}
}

func TestMaskPhoneDisplayUsesLocalFormat(t *testing.T) {
	masked := MaskPhoneDisplay("+66812345678")

	if masked != "081-XXX-5678" {
		t.Fatalf("expected 081-XXX-5678, got %s", masked)
	}
}

func TestMaskPhoneNeverRevealsShortInput(t *testing.T) {
	if masked := MaskPhone("1234"); masked != "XXXX" {
		t.Fatalf("expected XXXX, got %s", masked)
	}
}

func TestPhoneLast4(t *testing.T) {
	if last := PhoneLast4("+66812345678"); last != "5678" {
		t.Fatalf("expected 5678, got %s", last)
	}
}
