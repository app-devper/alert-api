package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func NewGatewayHost(allowedHosts string) gin.HandlerFunc {
	allowed := map[string]bool{}
	for _, host := range strings.Split(allowedHosts, ",") {
		host = strings.ToLower(strings.TrimSpace(host))
		if host != "" {
			allowed[host] = true
		}
	}
	if len(allowed) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Host"), ",")[0])
		if !allowed[strings.ToLower(forwarded)] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "direct access is not allowed"})
			return
		}
		c.Next()
	}
}
