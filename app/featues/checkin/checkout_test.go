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
	checkoutFn func(clientId, branchId string, id primitive.ObjectID) error
}

func (s *checkInStub) Checkout(clientId, branchId string, id primitive.ObjectID, _ time.Time, _ string) error {
	return s.checkoutFn(clientId, branchId, id)
}

func TestSelfCheckoutIsIdempotentAndScopedToTheCheckInsBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var gotBranch string
	repo := &domain.Repository{CheckIn: &checkInStub{checkoutFn: func(_, branchId string, _ primitive.ObjectID) error {
		gotBranch = branchId
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
	if gotBranch != "B7" {
		t.Fatalf("Checkout was called with Branch %q, want the Check-in's B7", gotBranch)
	}
}
