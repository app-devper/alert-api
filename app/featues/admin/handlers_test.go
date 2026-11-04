package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestUpsertPermissionRejectsAWhitespaceBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/admin/permissions",
		strings.NewReader(`{"userId":"u1","branchId":"   ","active":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	handleUpsertPermission(ctx, nil)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "AD-400-002") {
		t.Fatalf("expected AD-400-002, got %s", w.Body.String())
	}
}
