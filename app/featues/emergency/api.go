package emergency

import (
	"net/http"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/data/repositories"
	"alert/app/domain"
	"alert/app/domain/request"
	"alert/app/domain/staff"
	"alert/middlewares"

	"github.com/app-devper/um-api/sessionclient"
	"github.com/gin-gonic/gin"
)

func ApplyEmergencyAPI(route *gin.RouterGroup, repository *domain.Repository) {
	r := route.Group("emergency",
		middlewares.RequireSession(repository.Auth),
		middlewares.RequireTenant(),
		middlewares.RequireBranch(repository.StaffPermission),
		repository.Auth.AtLeast(sessionclient.RoleUser),
	)

	r.GET("/preview", func(ctx *gin.Context) {
		handlePreview(ctx, repository)
	})

	r.POST("/trigger", func(ctx *gin.Context) {
		handleTrigger(ctx, repository)
	})

	r.GET("/active", func(ctx *gin.Context) {
		s := staff.Current(ctx)
		event, err := repository.EmergencyEvent.GetLatestOpenEvent(s.Scope)
		if err != nil {
			errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
			return
		}
		response.Ok(ctx, event)
	})
}

func handlePreview(ctx *gin.Context, repository *domain.Repository) {
	s := staff.Current(ctx)
	clientId, branchId := s.Scope.ClientId, s.Scope.BranchId
	eventType := ctx.Query("eventType")
	if !constant.IsValidEventType(eventType) {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_001, "invalid eventType")
		return
	}
	if !s.MayTrigger(eventType) {
		errcode.Abort(ctx, http.StatusForbidden, errcode.EM_FORBIDDEN_001, "not allowed to trigger this event type")
		return
	}
	template, err := repository.MessageTemplate.GetActiveTemplateByCode(clientId, eventType)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.TP_NOT_FOUND_001, "no active template for event type")
		return
	}
	setting, err := repository.BranchSetting.GetSetting(clientId, branchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	now := time.Now()
	recipientCount, err := repository.CheckIn.CountActive(s.Scope, now)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	cooldown := cooldownStatus(repository, s.Scope, eventType, setting.CooldownSeconds, now)
	response.Ok(ctx, gin.H{
		"eventType":         eventType,
		"template":          template,
		"recipientCount":    recipientCount,
		"confirmMethod":     setting.ConfirmMethod,
		"cooldownRemaining": cooldown,
	})
}

func cooldownStatus(repository *domain.Repository, scope repositories.Scope, eventType string, cooldownSeconds int, now time.Time) float64 {
	if alerting.IsCooldownExempt(eventType) {
		return 0
	}
	latest, err := repository.EmergencyEvent.GetLatestEvent(scope, eventType)
	if err != nil || latest == nil {
		return 0
	}
	return alerting.CooldownRemaining(&latest.SentAt, now, cooldownSeconds).Seconds()
}

func handleTrigger(ctx *gin.Context, repository *domain.Repository) {
	s := staff.Current(ctx)
	var req request.TriggerAlert
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_001, err.Error())
		return
	}
	if !constant.IsValidEventType(req.EventType) {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_001, "invalid eventType")
		return
	}
	if !s.MayTrigger(req.EventType) {
		auditTrigger(ctx, repository, req.EventType, constant.ResultFailed, "no permission")
		errcode.Abort(ctx, http.StatusForbidden, errcode.EM_FORBIDDEN_001, "not allowed to trigger this event type")
		return
	}
	if req.EventType == constant.EventTest {
		handleTestAlert(ctx, repository, req)
		return
	}
	handleRealAlert(ctx, repository, req)
}
