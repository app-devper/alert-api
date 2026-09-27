package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"alert/app/core/constant"

	"github.com/gin-gonic/gin"
)

func runRequireAuthorization(role string, allowed ...string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Set("Role", role)
	RequireAuthorization(allowed...)(c)
	return c, w
}

// UM issues USER, MANAGER, ADMIN and SUPER; every one of them is a member.
func TestMemberRolesAdmitEveryUMRole(t *testing.T) {
	for _, role := range []string{"USER", "MANAGER", "ADMIN", "SUPER"} {
		if c, w := runRequireAuthorization(role, constant.MemberRoles...); c.IsAborted() {
			t.Errorf("%s: expected access, got %d %s", role, w.Code, w.Body.String())
		}
	}
}

func TestMemberRolesRejectUnknownRole(t *testing.T) {
	for _, role := range []string{"STAFF", ""} {
		if c, w := runRequireAuthorization(role, constant.MemberRoles...); !c.IsAborted() || w.Code != http.StatusForbidden {
			t.Errorf("%q: expected 403, got %d", role, w.Code)
		}
	}
}
