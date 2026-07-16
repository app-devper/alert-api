package errcode

import (
	"alert/app/core/response"

	"github.com/gin-gonic/gin"
)

func Abort(ctx *gin.Context, httpStatus int, code string, msg string) {
	response.Abort(ctx, httpStatus, code, msg)
}
