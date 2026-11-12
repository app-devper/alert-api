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
