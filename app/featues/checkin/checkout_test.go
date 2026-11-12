package checkin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type checkInStub struct {
	repositories.ICheckIn
	checkoutFn func(scope repositories.Scope, id primitive.ObjectID) error
}

func (s *checkInStub) Checkout(scope repositories.Scope, id primitive.ObjectID, _ time.Time, _ string) error {
	return s.checkoutFn(scope, id)
}

func TestSelfCheckoutIsIdempotentAndScopedToTheCheckInsBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotScope repositories.Scope
	repo := &domain.Repository{CheckIn: &checkInStub{checkoutFn: func(scope repositories.Scope, _ primitive.ObjectID) error {
		gotScope = scope
		return mongo.ErrNoDocuments
	}}}
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/public/me/checkout", nil)
	ctx.Set("CheckIn", entities.CheckIn{Id: primitive.NewObjectID(), ClientId: "c01", BranchId: "B7"})

	handleSelfCheckout(ctx, repo)

	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "CHECKED_OUT") {
		t.Fatalf("expected 200 CHECKED_OUT, got %d %s", w.Code, w.Body.String())
	}
	if want := (repositories.Scope{ClientId: "c01", BranchId: "B7"}); gotScope != want {
		t.Fatalf("Checkout was called with %+v, want the Check-in's %+v", gotScope, want)
	}
}

type scopeRecordingCheckIn struct {
	repositories.ICheckIn
	scopes []repositories.Scope
}

func (s *scopeRecordingCheckIn) SetPushSubscription(scope repositories.Scope, _ primitive.ObjectID, _ *entities.PushSubscription) error {
	s.scopes = append(s.scopes, scope)
	return nil
}

func (s *scopeRecordingCheckIn) ClearPushSubscription(scope repositories.Scope, _ primitive.ObjectID) error {
	s.scopes = append(s.scopes, scope)
	return nil
}

func (s *scopeRecordingCheckIn) DeleteCheckIn(scope repositories.Scope, _ primitive.ObjectID) error {
	s.scopes = append(s.scopes, scope)
	return nil
}

type auditStub struct {
	repositories.IAuditLog
	recorded []entities.AuditLog
}

func (s *auditStub) Record(log entities.AuditLog) { s.recorded = append(s.recorded, log) }

func TestCustomerWritesAreScopedToTheCheckInsBranch(t *testing.T) {
	cases := []struct {
		name string
		body string
		call func(*gin.Context, *domain.Repository)
	}{
		{"push subscribe", `{"endpoint":"https://push.example/1","p256dh":"k","auth":"a"}`, handlePushSubscribe},
		{"push unsubscribe", "", handlePushUnsubscribe},
		{"withdraw consent", "", handleWithdrawConsent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			checkIns, audit := &scopeRecordingCheckIn{}, &auditStub{}
			repo := &domain.Repository{CheckIn: checkIns, AuditLog: audit}
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/public/me", strings.NewReader(c.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			ctx.Set("CheckIn", entities.CheckIn{Id: primitive.NewObjectID(), ClientId: "c01", BranchId: "B7"})

			c.call(ctx, repo)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d %s", w.Code, w.Body.String())
			}
			want := repositories.Scope{ClientId: "c01", BranchId: "B7"}
			if len(checkIns.scopes) != 1 || checkIns.scopes[0] != want {
				t.Fatalf("repository called with %+v, want one call with %+v", checkIns.scopes, want)
			}
		})
	}
}
