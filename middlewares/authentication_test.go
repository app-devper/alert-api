package middlewares

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert/app/core/config"
	"alert/app/core/errcode"
	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain/staff"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type staffPermissionRepoStub struct {
	repositories.IStaffPermission
	getByUserIdFn func(clientId string, userId string) (entities.StaffPermission, error)
}

func (s *staffPermissionRepoStub) GetByUserId(clientId string, userId string) (entities.StaffPermission, error) {
	return s.getByUserIdFn(clientId, userId)
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

func TestRequireBranchRefusesCallerWithoutPermission(t *testing.T) {
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

	if !ctx.IsAborted() {
		t.Fatalf("expected a caller without an Alert Permission to be refused")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
	if !strings.Contains(w.Body.String(), errcode.AU_FORBIDDEN_002) {
		t.Fatalf("expected body to contain %s, got %s", errcode.AU_FORBIDDEN_002, w.Body.String())
	}
	if got := staff.Current(ctx).Scope.BranchId; got != "" {
		t.Fatalf("expected no Branch to be set, got %q", got)
	}
}

func TestRequireBranchRefusesPermissionWithoutBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{Active: true, BranchId: ""}, nil
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if !ctx.IsAborted() {
		t.Fatalf("expected a Permission without a Branch to be refused")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, w.Code)
	}
	if !strings.Contains(w.Body.String(), errcode.AU_FORBIDDEN_002) {
		t.Fatalf("expected body to contain %s, got %s", errcode.AU_FORBIDDEN_002, w.Body.String())
	}
}

func TestRequireBranchAnswers503WhenTheStoreFails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{}, errors.New("down")
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "001")

	RequireBranch(permissionRepo)(ctx)

	if !ctx.IsAborted() {
		t.Fatalf("expected the middleware to abort when the store fails")
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
	if !strings.Contains(w.Body.String(), errcode.AU_UNAVAILABLE_002) {
		t.Fatalf("expected body to contain %s, got %s", errcode.AU_UNAVAILABLE_002, w.Body.String())
	}
}

func TestRequireBranchUsesPermissionBranchWhenFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionRepo := &staffPermissionRepoStub{
		getByUserIdFn: func(clientId string, userId string) (entities.StaffPermission, error) {
			return entities.StaffPermission{
				ClientId: "c01", BranchId: "B1", Active: true, AllowedEventTypes: []string{"FIRE"},
			}, nil
		},
	}

	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	ctx.Set("UserId", "user-1")
	ctx.Set("ClientId", "c01")
	ctx.Set("Role", "USER")

	RequireBranch(permissionRepo)(ctx)

	if ctx.IsAborted() {
		t.Fatalf("expected middleware to continue")
	}
	s := staff.Current(ctx)
	if s.Scope != (repositories.Scope{ClientId: "c01", BranchId: "B1"}) || s.UserId != "user-1" {
		t.Fatalf("Staff not resolved from the Permission: %+v", s)
	}
	if !s.MayTrigger("FIRE") || s.MayTrigger("EVACUATE") {
		t.Fatal("Staff does not carry the Permission's Event Types")
	}
	if _, exists := ctx.Get("BranchId"); exists {
		t.Fatal("the BranchId string key must be gone")
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

func TestRequireTenantRefusesClientIdThatCannotNameADatabase(t *testing.T) {
	for clientId, want := range map[string]int{"001": http.StatusOK, "": http.StatusUnauthorized, "../x": http.StatusUnauthorized} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		c.Set("ClientId", clientId)
		RequireTenant()(c)
		if got := map[bool]int{true: http.StatusUnauthorized, false: http.StatusOK}[c.IsAborted()]; got != want {
			t.Errorf("clientId %q: got %d, want %d", clientId, got, want)
		}
	}
}
