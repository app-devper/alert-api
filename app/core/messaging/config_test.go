package messaging

import "testing"

func TestMergedOverPrefersTenantValues(t *testing.T) {
	tenant := ProviderConfig{SmsSenderId: "SHOPA", LineChannelToken: "tenant-token"}
	fallback := ProviderConfig{SmsSenderId: "SHARED", SmsApiKey: "shared-key", LineChannelToken: "shared-token"}

	merged := tenant.MergedOver(fallback)

	if merged.SmsSenderId != "SHOPA" {
		t.Fatalf("expected tenant sender id, got %s", merged.SmsSenderId)
	}
	if merged.LineChannelToken != "tenant-token" {
		t.Fatalf("expected tenant line token, got %s", merged.LineChannelToken)
	}
	if merged.SmsApiKey != "shared-key" {
		t.Fatalf("expected fallback api key, got %s", merged.SmsApiKey)
	}
}

func TestMergedOverFallsBackWhenTenantEmpty(t *testing.T) {
	merged := ProviderConfig{}.MergedOver(ProviderConfig{SmsApiUrl: "https://sms.example", SmsWebhookSecret: "sec"})

	if merged.SmsApiUrl != "https://sms.example" || merged.SmsWebhookSecret != "sec" {
		t.Fatalf("expected fallback values, got %+v", merged)
	}
}

func TestHasSmsRequiresUrlKeyAndSender(t *testing.T) {
	if (ProviderConfig{SmsApiUrl: "u", SmsApiKey: "k"}).HasSms() {
		t.Fatal("missing sender id must not count as configured")
	}
	if !(ProviderConfig{SmsApiUrl: "u", SmsApiKey: "k", SmsSenderId: "S"}).HasSms() {
		t.Fatal("expected configured sms")
	}
}

func TestHasLineRequiresToken(t *testing.T) {
	if (ProviderConfig{}).HasLine() {
		t.Fatal("empty token must not count as configured")
	}
	if !(ProviderConfig{LineChannelToken: "t"}).HasLine() {
		t.Fatal("expected configured line")
	}
}
