package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func signBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyLineSignatureAcceptsValidSignature(t *testing.T) {
	body := []byte(`{"events":[]}`)

	if !VerifyLineSignature("secret", body, signBody("secret", body)) {
		t.Fatal("expected valid signature to pass")
	}
}

func TestVerifyLineSignatureRejectsWrongSecret(t *testing.T) {
	body := []byte(`{"events":[]}`)

	if VerifyLineSignature("secret", body, signBody("other", body)) {
		t.Fatal("expected wrong secret to fail")
	}
}

func TestVerifyLineSignatureRejectsMissingSignature(t *testing.T) {
	if VerifyLineSignature("secret", []byte("{}"), "") {
		t.Fatal("expected missing signature to fail")
	}
}

func TestVerifyLineSignatureSkipsWhenSecretUnset(t *testing.T) {
	if !VerifyLineSignature("", []byte("{}"), "") {
		t.Fatal("expected pass-through when secret not configured")
	}
}
