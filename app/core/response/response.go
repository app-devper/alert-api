package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Meta struct {
	Total int64 `json:"total"`
	Page  int64 `json:"page"`
	Limit int64 `json:"limit"`
}

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   *ErrorBody  `json:"error"`
	Meta    *Meta       `json:"meta,omitempty"`
}

func Ok(ctx *gin.Context, data interface{}) {
	ctx.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

func OkWithMeta(ctx *gin.Context, data interface{}, meta Meta) {
	ctx.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: &meta})
}

func Abort(ctx *gin.Context, httpStatus int, code string, message string) {
	ctx.AbortWithStatusJSON(httpStatus, Envelope{Success: false, Error: &ErrorBody{Code: code, Message: message}})
}
