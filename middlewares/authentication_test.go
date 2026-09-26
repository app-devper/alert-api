package middlewares

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"alert/app/core/config"
	"alert/app/core/errcode"
	"alert/app/data/entities"
	"alert/app/data/repositories"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/mongo"
)

type staffPermissionRepoStub struct {
	repositories.IStaffPermission
	getByUserIdFn func(clientId string, userId string) (entities.StaffPermission, error)
}

func (s *staffPermissionRepoStub) GetByUserId(clientId string, userId string) (entities.StaffPermission, error) {
	return s.getByUserIdFn(clientId, userId)
}

func signToken(t *testing.T, secretKey string, claims AccessClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secretKey))
	if err != nil {
		t.Fatalf("unexpected error signing token: %v", err)
	}
	return token
}

func testConfig() *config.Config {
	return &config.Config{SecretKey: "test-secret-key", System: "ALERT"}
}

func requestWithAuth(bearer string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	return req
}

func runMiddleware(handler gin.HandlerFunc, req *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = req
	handler(ctx)
	return w
}

func TestRequireAuthenticatedAcceptsValidToken(t *testing.T) {
	cfg := testConfig()
	claims := AccessClaims{
		Role: "ADMIN", System: cfg.System, ClientId: "001",
		RegisteredClaims: jwt.RegisteredClaims{ID: "session-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	req := requestWithAuth(signToken(t, cfg.SecretKey, claims))

	w := runMiddleware(RequireAuthenticated(cfg), req)

	if w.Code != 0 && w.Code != http.StatusOK {
		t.Fatalf("expected middleware to continue, got status %d body %s", w.Code, w.Body.String())
	}
}

func TestRequireAuthenticatedRejectsMissingHeader(t *testing.T) {
	w := runMiddleware(RequireAuthenticated(testConfig()), requestWithAuth(""))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthenticatedRejectsWrongSigningSecret(t *testing.T) {
	cfg := testConfig()
	claims := AccessClaims{
		Role: "ADMIN", System: cfg.System, ClientId: "001",
		RegisteredClaims: jwt.RegisteredClaims{ID: "session-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	req := requestWithAuth(signToken(t, "wrong-secret", claims))

	w := runMiddleware(RequireAuthenticated(cfg), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthenticatedRejectsSystemMismatch(t *testing.T) {
	cfg := testConfig()
	claims := AccessClaims{
		Role: "ADMIN", System: "POS", ClientId: "001",
		RegisteredClaims: jwt.RegisteredClaims{ID: "session-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	req := requestWithAuth(signToken(t, cfg.SecretKey, claims))

	w := runMiddleware(RequireAuthenticated(cfg), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireAuthenticatedRejectsPinnedClientIdMismatch(t *testing.T) {
	cfg := testConfig()
	cfg.ClientId = "001"
	claims := AccessClaims{
		Role: "ADMIN", System: cfg.System, ClientId: "002",
		RegisteredClaims: jwt.RegisteredClaims{ID: "session-1", ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	}
	req := requestWithAuth(signToken(t, cfg.SecretKey, claims))

	w := runMiddleware(RequireAuthenticated(cfg), req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequireBranchFallsBackToHQOnlyWhenPermissionNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{}, mongo.ErrNoDocuments
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if ctx.IsAborted() {
		t.Fatalf("expected fallback to continue, got aborted with status %d", w.Code)
	}
	if got := ctx.GetString("BranchId"); got != "HQ" {
		t.Fatalf("expected fallback branch HQ, got %q", got)
	}
}

func TestRequireBranchAbortsOnLookupErrorWithoutFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{}, errors.New("mongo unavailable")
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if !ctx.IsAborted() {
		t.Fatalf("expected middleware to abort on lookup error, not fall back to HQ")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequireBranchUsesPermissionBranchWhenFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{
				BranchId: "BRANCH-2", Active: true, AllowedEventTypes: []string{"FIRE"},
			}, nil
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if ctx.IsAborted() {
		t.Fatalf("expected middleware to continue")
	}
	if got := ctx.GetString("BranchId"); got != "BRANCH-2" {
		t.Fatalf("expected BRANCH-2, got %q", got)
	}
}

func TestRequireBranchRejectsInactivePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{BranchId: "BRANCH-2", Active: false}, nil
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if !ctx.IsAborted() || w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for inactive permission, got aborted=%v status=%d", ctx.IsAborted(), w.Code)
	}
}

type sessionStub struct {
	userId    string
	err       error
	gotSystem string
	gotMethod string
}

func (s *sessionStub) Authorize(_ context.Context, _ string, system, method string) (string, error) {
	s.gotSystem, s.gotMethod = system, method
	return s.userId, s.err
}

func runRequireSession(stub *sessionStub, method string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(method, "/", nil)
	ctx.Set("SessionId", "session-1")
	ctx.Set("System", "ALERT")
	RequireSession(stub)(ctx)
	return ctx, w
}

func TestRequireSessionSetsUserIdFromLiveSession(t *testing.T) {
	stub := &sessionStub{userId: "user-1"}
	ctx, w := runRequireSession(stub, http.MethodGet)
	if ctx.IsAborted() || ctx.GetString("UserId") != "user-1" {
		t.Fatalf("expected UserId user-1, got %q (status %d)", ctx.GetString("UserId"), w.Code)
	}
	if stub.gotSystem != "ALERT" || stub.gotMethod != http.MethodGet {
		t.Fatalf("expected token system and request method, got %q %q", stub.gotSystem, stub.gotMethod)
	}
}

func TestRequireSessionRejectsRevokedSession(t *testing.T) {
	_, w := runRequireSession(&sessionStub{err: sessionclient.ErrSessionRejected}, http.MethodGet)
	if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), errcode.AU_UNAUTHORIZED_005) {
		t.Fatalf("expected 401 %s, got %d %s", errcode.AU_UNAUTHORIZED_005, w.Code, w.Body.String())
	}
}

func TestRequireSessionReturns503WhenSessionStoreUnavailable(t *testing.T) {
	_, w := runRequireSession(&sessionStub{err: sessionclient.ErrUnavailable}, http.MethodPost)
	if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), errcode.AU_UNAVAILABLE_001) {
		t.Fatalf("expected 503 %s, got %d %s", errcode.AU_UNAVAILABLE_001, w.Code, w.Body.String())
	}
}
