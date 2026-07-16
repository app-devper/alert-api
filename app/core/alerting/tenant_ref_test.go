package alerting

import "testing"

func TestComposeAndSplitTenantRef(t *testing.T) {
	ref := ComposeTenantRef("001", "abc123")

	clientId, value, err := SplitTenantRef(ref)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientId != "001" || value != "abc123" {
		t.Fatalf("expected 001/abc123, got %s/%s", clientId, value)
	}
}

func TestSplitTenantRefKeepsDotsInValue(t *testing.T) {
	clientId, value, err := SplitTenantRef("002.a.b.c")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clientId != "002" || value != "a.b.c" {
		t.Fatalf("expected 002/a.b.c, got %s/%s", clientId, value)
	}
}

func TestSplitTenantRefRejectsMalformed(t *testing.T) {
	for _, ref := range []string{"", "no-separator", ".value", "client."} {
		if _, _, err := SplitTenantRef(ref); err == nil {
			t.Fatalf("expected error for %q", ref)
		}
	}
}
