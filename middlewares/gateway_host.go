package middlewares

import (
	"net/http"

	"alert/app/core/errcode"

	"github.com/app-devper/um-api/servicekit/gateway"
	"github.com/app-devper/um-api/servicekit/gateway/gingateway"
	"github.com/gin-gonic/gin"
)

// NewGatewayHost refuses requests that did not come through the gateway
// (um-api servicekit, its ADR-0007), in this service's error envelope.
func NewGatewayHost(allowedHosts string) gin.HandlerFunc {
	return gingateway.Middleware(gateway.ParseHosts(allowedHosts), func(c *gin.Context) {
		errcode.Abort(c, http.StatusForbidden, errcode.SY_FORBIDDEN_001, gateway.Message)
	})
}
