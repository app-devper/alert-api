package middlewares

import (
	"net/http"

	"alert/app/core/errcode"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func NewRecovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(ctx *gin.Context, recovered interface{}) {
		logrus.Error("panic recovered: ", recovered)
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.SY_INTERNAL_001, "internal server error")
	})
}
