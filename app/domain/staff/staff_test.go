package staff

import (
	"net/http/httptest"
	"testing"

	"alert/app/core/constant"
	"alert/app/data/entities"
	"alert/app/data/repositories"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func TestMayTriggerFollowsThePermissionBelowManager(t *testing.T) {
	permission := entities.StaffPermission{ClientId: "c01", BranchId: "B1", AllowedEventTypes: []string{"FIRE"}}
	cases := []struct {
		name      string
		role      sessionclient.Role
		eventType string
		want      bool
	}{
		{"user may trigger a listed type", sessionclient.RoleUser, "FIRE", true},
		{"user may not trigger an unlisted type", sessionclient.RoleUser, "EVACUATE", false},
		{"manager may trigger anything", sessionclient.RoleManager, "EVACUATE", true},
		{"admin may trigger anything", sessionclient.RoleAdmin, "ALL_CLEAR", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := FromPermission("u1", c.role, permission).MayTrigger(c.eventType); got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestTestModeAndCooldownOverrideNeedManager(t *testing.T) {
	permission := entities.StaffPermission{ClientId: "c01", BranchId: "B1", AllowedEventTypes: []string{"FIRE", "TEST"}}
	user := FromPermission("u1", sessionclient.RoleUser, permission)
	manager := FromPermission("m1", sessionclient.RoleManager, permission)
	if user.MayTest() || user.MayOverrideCooldown() {
		t.Fatal("a USER may not test or override the Cooldown, whatever the Permission lists")
	}
	if !manager.MayTest() || !manager.MayOverrideCooldown() {
		t.Fatal("a MANAGER may test and override the Cooldown")
	}
}

func TestFromPermissionCarriesTheBranchAsScope(t *testing.T) {
	s := FromPermission("u1", sessionclient.RoleUser, entities.StaffPermission{ClientId: "c01", BranchId: "B1"})
	if s.Scope != (repositories.Scope{ClientId: "c01", BranchId: "B1"}) || s.UserId != "u1" {
		t.Fatalf("got %+v", s)
	}
	entry := s.AuditEntry(constant.ActionCheckoutCustomer, constant.ResultSuccess, bson.M{"checkInId": "x"})
	if entry.ClientId != "c01" || entry.BranchId != "B1" || entry.Actor != "u1" || entry.Action != constant.ActionCheckoutCustomer || entry.Result != constant.ResultSuccess {
		t.Fatalf("audit entry not filled from Staff: %+v", entry)
	}
}

func TestCurrentIsZeroWithoutRequireBranch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	if got := Current(ctx); got.UserId != "" || got.Scope != (repositories.Scope{}) {
		t.Fatalf("got %+v, want the zero Staff", got)
	}
	Set(ctx, FromPermission("u1", sessionclient.RoleUser, entities.StaffPermission{ClientId: "c01", BranchId: "B1"}))
	if Current(ctx).Scope.BranchId != "B1" {
		t.Fatal("Set/Current do not round-trip")
	}
}

func TestZeroStaffMayDoNothing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())

	zero := Current(ctx)

	if zero.MayTrigger("FIRE") || zero.MayTest() || zero.MayOverrideCooldown() {
		t.Fatalf("the zero Staff must fail closed: trigger %v, test %v, override %v",
			zero.MayTrigger("FIRE"), zero.MayTest(), zero.MayOverrideCooldown())
	}
}
