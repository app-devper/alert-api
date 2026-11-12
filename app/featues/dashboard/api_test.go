package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain"
	"alert/app/domain/staff"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type checkInStub struct {
	repositories.ICheckIn
	checkoutFn func(clientId, branchId string, id primitive.ObjectID) error
}

func (s *checkInStub) Checkout(clientId, branchId string, id primitive.ObjectID, _ time.Time, _ string) error {
	return s.checkoutFn(clientId, branchId, id)
}

type eventStub struct {
	repositories.IEmergencyEvent
	getFn func(clientId, branchId string, id primitive.ObjectID) (entities.EmergencyEvent, error)
}

func (s *eventStub) GetEventById(clientId, branchId string, id primitive.ObjectID) (entities.EmergencyEvent, error) {
	return s.getFn(clientId, branchId, id)
}

type auditStub struct {
	repositories.IAuditLog
	recorded []entities.AuditLog
}

func (s *auditStub) Record(log entities.AuditLog) { s.recorded = append(s.recorded, log) }

func staffContext(t *testing.T, method, path, id string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(method, path, nil)
	ctx.Params = gin.Params{{Key: "id", Value: id}}
	ctx.Set("ClientId", "c01")
	ctx.Set("UserId", "u1")
	staff.Set(ctx, staff.FromPermission("u1", sessionclient.RoleUser, entities.StaffPermission{ClientId: "c01", BranchId: "B1"}))
	return ctx, w
}

func TestStaffCheckoutOfAForeignBranchIs404AndNotAudited(t *testing.T) {
	id := primitive.NewObjectID()
	audit := &auditStub{}
	var gotBranch string
	repo := &domain.Repository{
		CheckIn: &checkInStub{checkoutFn: func(_, branchId string, _ primitive.ObjectID) error {
			gotBranch = branchId
			return mongo.ErrNoDocuments
		}},
		AuditLog: audit,
	}
	ctx, w := staffContext(t, http.MethodPost, "/dashboard/check-ins/"+id.Hex()+"/checkout", id.Hex())

	handleStaffCheckout(ctx, repo)

	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), errcode.CK_NOT_FOUND_001) {
		t.Fatalf("expected 404 %s, got %d %s", errcode.CK_NOT_FOUND_001, w.Code, w.Body.String())
	}
	if gotBranch != "B1" {
		t.Fatalf("Checkout was called with Branch %q, want the Staff's B1", gotBranch)
	}
	if len(audit.recorded) != 0 {
		t.Fatalf("a miss was audited: %+v", audit.recorded)
	}
}

func TestStaffCheckoutInOwnBranchIsAuditedOnce(t *testing.T) {
	id := primitive.NewObjectID()
	audit := &auditStub{}
	repo := &domain.Repository{
		CheckIn:  &checkInStub{checkoutFn: func(string, string, primitive.ObjectID) error { return nil }},
		AuditLog: audit,
	}
	ctx, w := staffContext(t, http.MethodPost, "/dashboard/check-ins/"+id.Hex()+"/checkout", id.Hex())

	handleStaffCheckout(ctx, repo)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d %s", w.Code, w.Body.String())
	}
	if len(audit.recorded) != 1 || audit.recorded[0].BranchId != "B1" || audit.recorded[0].Action != constant.ActionCheckoutCustomer {
		t.Fatalf("expected one checkout Audit entry for B1, got %+v", audit.recorded)
	}
}

func TestEventDetailOfAForeignBranchIs404(t *testing.T) {
	id := primitive.NewObjectID()
	var gotBranch string
	repo := &domain.Repository{EmergencyEvent: &eventStub{getFn: func(_, branchId string, _ primitive.ObjectID) (entities.EmergencyEvent, error) {
		gotBranch = branchId
		return entities.EmergencyEvent{}, mongo.ErrNoDocuments
	}}}
	ctx, w := staffContext(t, http.MethodGet, "/dashboard/events/"+id.Hex(), id.Hex())

	handleEventDetail(ctx, repo)

	if w.Code != http.StatusNotFound || !strings.Contains(w.Body.String(), errcode.EM_NOT_FOUND_001) {
		t.Fatalf("expected 404 %s, got %d %s", errcode.EM_NOT_FOUND_001, w.Code, w.Body.String())
	}
	if gotBranch != "B1" {
		t.Fatalf("GetEventById was called with Branch %q, want B1", gotBranch)
	}
}
