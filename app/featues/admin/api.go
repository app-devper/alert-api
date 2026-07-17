package admin

import (
	"net/http"

	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/domain"
	"alert/middlewares"

	"github.com/gin-gonic/gin"
)

func ApplyAdminAPI(route *gin.RouterGroup, repository *domain.Repository) {
	r := route.Group("admin",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(repository.Session),
		middlewares.RequireBranch(repository.StaffPermission),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN),
	)

	r.GET("/templates", func(ctx *gin.Context) {
		templates, err := repository.MessageTemplate.GetTemplates(ctx.GetString("ClientId"))
		if err != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.TP_INTERNAL_001, err.Error())
			return
		}
		response.Ok(ctx, templates)
	})

	r.POST("/templates", func(ctx *gin.Context) {
		handleCreateTemplate(ctx, repository)
	})

	r.PUT("/templates/:id", func(ctx *gin.Context) {
		handleUpdateTemplate(ctx, repository)
	})

	r.GET("/settings", func(ctx *gin.Context) {
		setting, err := repository.BranchSetting.GetSetting(ctx.GetString("ClientId"), ctx.GetString("BranchId"))
		if err != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
			return
		}
		response.Ok(ctx, gin.H{"setting": setting, "hasPin": setting.HasPin()})
	})

	r.PUT("/settings", func(ctx *gin.Context) {
		handleUpdateSetting(ctx, repository)
	})

	r.PUT("/settings/pin", func(ctx *gin.Context) {
		handleSetPin(ctx, repository)
	})

	r.GET("/permissions", func(ctx *gin.Context) {
		permissions, err := repository.StaffPermission.GetPermissions(ctx.GetString("ClientId"))
		if err != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
			return
		}
		response.Ok(ctx, permissions)
	})

	r.PUT("/permissions", func(ctx *gin.Context) {
		handleUpsertPermission(ctx, repository)
	})

	r.GET("/qr", func(ctx *gin.Context) {
		tokens, err := repository.QrToken.GetQrTokens(ctx.GetString("ClientId"), ctx.Query("branchId"))
		if err != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
			return
		}
		response.Ok(ctx, tokens)
	})

	r.POST("/qr", func(ctx *gin.Context) {
		handleCreateQr(ctx, repository)
	})

	r.POST("/qr/:id/revoke", func(ctx *gin.Context) {
		handleRevokeQr(ctx, repository)
	})

	r.GET("/qr/:id/image", func(ctx *gin.Context) {
		handleQrImage(ctx, repository)
	})

	r.GET("/sms-credit", func(ctx *gin.Context) {
		handleSmsCredit(ctx, repository)
	})

	r.GET("/messaging-config", func(ctx *gin.Context) {
		handleGetMessagingConfig(ctx, repository)
	})

	r.PUT("/messaging-config", func(ctx *gin.Context) {
		handleUpdateMessagingConfig(ctx, repository)
	})
}
