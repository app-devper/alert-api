package middlewares

import (
	"net/http"

	"alert/app/core/errcode"

	"github.com/gin-gonic/gin"
)

func NoRoute() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		errcode.Abort(ctx, http.StatusNotFound, errcode.SY_NOT_FOUND_001, "route not found")
	}
}
