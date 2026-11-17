package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain"
	"alert/app/domain/staff"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
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

type auditStub struct {
	repositories.IAuditLog
	recorded []entities.AuditLog
}

func (s *auditStub) Record(log entities.AuditLog) { s.recorded = append(s.recorded, log) }

type permissionStub struct {
	repositories.IStaffPermission
	upserted []entities.StaffPermission
}

func (s *permissionStub) UpsertPermission(permission entities.StaffPermission) error {
	s.upserted = append(s.upserted, permission)
	return nil
}

func adminContext(w *httptest.ResponseRecorder, body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/admin/permissions", strings.NewReader(body))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("ClientId", "c01")
	ctx.Set("UserId", "admin1")
	ctx.Set("Role", string(sessionclient.RoleAdmin))
	return ctx
}

func TestUpsertPermissionAuditFallsBackToTheSessionWithoutStaff(t *testing.T) {
	audit, permissions := &auditStub{}, &permissionStub{}
	repo := &domain.Repository{AuditLog: audit, StaffPermission: permissions}
	w := httptest.NewRecorder()
	ctx := adminContext(w, `{"userId":"u2","branchId":"B3","active":true}`)

	handleUpsertPermission(ctx, repo)

	if w.Code != http.StatusOK || len(permissions.upserted) != 1 {
		t.Fatalf("expected 200 and one upsert, got %d %s", w.Code, w.Body.String())
	}
	if len(audit.recorded) != 1 {
		t.Fatalf("expected one Audit entry, got %+v", audit.recorded)
	}
	entry := audit.recorded[0]
	if entry.ClientId != "c01" || entry.Actor != "admin1" || entry.BranchId != "" || entry.Action != constant.ActionUpdatePermission {
		t.Fatalf("Audit entry not taken from the session keys: %+v", entry)
	}
}

func TestAuditAdminUsesTheStaffsClientBranchAndActor(t *testing.T) {
	audit := &auditStub{}
	repo := &domain.Repository{AuditLog: audit}
	ctx := adminContext(httptest.NewRecorder(), "")
	ctx.Set("ClientId", "session-client")
	ctx.Set("UserId", "session-user")
	staff.Set(ctx, staff.FromPermission("staff1", sessionclient.RoleAdmin, entities.StaffPermission{ClientId: "c01", BranchId: "B7"}))

	auditAdmin(ctx, repo, constant.ActionUpdateSetting, bson.M{"retentionHours": 6})

	if len(audit.recorded) != 1 {
		t.Fatalf("expected one Audit entry, got %+v", audit.recorded)
	}
	entry := audit.recorded[0]
	if entry.ClientId != "c01" || entry.BranchId != "B7" || entry.Actor != "staff1" || entry.Action != constant.ActionUpdateSetting {
		t.Fatalf("Audit entry not taken from the Staff: %+v", entry)
	}
}
