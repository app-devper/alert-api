package staff

import "github.com/gin-gonic/gin"

// staffKey is the one gin key Staff lives under.
const staffKey = "alert.staff"

// Set stores s on the request context.
func Set(ctx *gin.Context, s Staff) { ctx.Set(staffKey, s) }

// Current returns the Staff set for this request, or the zero Staff when none was set.
func Current(ctx *gin.Context) Staff {
	s, _ := ctx.Get(staffKey)
	value, _ := s.(Staff)
	return value
}
