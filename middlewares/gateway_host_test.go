package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func gatewayRequest(t *testing.T, allowedHosts, forwardedHost string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(NewGatewayHost(allowedHosts))
	r.GET("/probe", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	if forwardedHost != "" {
		req.Header.Set("X-Forwarded-Host", forwardedHost)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestGatewayHostDisabledPassesWithoutHeader(t *testing.T) {
	if rec := gatewayRequest(t, "", ""); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGatewayHostAllowsMatchingForwardedHost(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app", "api.devper.app"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGatewayHostRejectsMissingHeader(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGatewayHostRejectsUnknownHost(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app", "evil.example.com"); rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestGatewayHostUsesFirstForwardedValue(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app", "api.devper.app, cdn.internal"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGatewayHostMatchesCaseInsensitive(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app", "API.Devper.App"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGatewayHostAllowsAnyConfiguredHost(t *testing.T) {
	if rec := gatewayRequest(t, "api.devper.app, devper-api.web.app", "devper-api.web.app"); rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
