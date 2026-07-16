package db

import "testing"

func TestDbNameForRootClientUsesPrefix(t *testing.T) {
	if name := DbNameFor("alert", "000"); name != "alert" {
		t.Fatalf("expected alert, got %s", name)
	}
}

func TestDbNameForTenantAppendsClientId(t *testing.T) {
	if name := DbNameFor("alert", "001"); name != "alert_001" {
		t.Fatalf("expected alert_001, got %s", name)
	}
}

func TestValidateClientIDAcceptsTypicalIds(t *testing.T) {
	for _, clientId := range []string{"000", "001", "abc", "shop-1", "a_b"} {
		if err := ValidateClientID(clientId); err != nil {
			t.Fatalf("expected %q to be valid: %v", clientId, err)
		}
	}
}

func TestValidateClientIDRejectsInvalidIds(t *testing.T) {
	for _, clientId := range []string{"", ".", "a.b", "-abc", "abc-", "a b", "$inject"} {
		if err := ValidateClientID(clientId); err == nil {
			t.Fatalf("expected %q to be rejected", clientId)
		}
	}
}
