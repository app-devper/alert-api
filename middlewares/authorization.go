package middlewares

import (
	"net/http"

	"alert/app/core/errcode"

	"github.com/gin-gonic/gin"
)

func RequireAuthorization(auths ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		role := ctx.GetString("Role")
		if role == "" {
			errcode.Abort(ctx, http.StatusForbidden, errcode.SY_FORBIDDEN_001, "Invalid request, restricted endpoint")
			return
		}
		for _, auth := range auths {
			if role == auth {
				ctx.Next()
				return
			}
		}
		errcode.Abort(ctx, http.StatusForbidden, errcode.SY_FORBIDDEN_002, "Don't have permission")
	}
}
