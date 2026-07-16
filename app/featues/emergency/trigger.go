package emergency

import (
	"fmt"
	"net/http"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/messaging"
	"alert/app/core/response"
	"alert/app/data/entities"
	"alert/app/domain"
	"alert/app/domain/request"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

func handleRealAlert(ctx *gin.Context, repository *domain.Repository, req request.TriggerAlert) {
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	now := time.Now()

	setting, err := repository.BranchSetting.GetSetting(clientId, branchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	if !verifyConfirmation(ctx, repository, setting, req) {
		return
	}
	if !passCooldown(ctx, repository, req, setting.CooldownSeconds, now) {
		return
	}

	template, err := repository.MessageTemplate.GetActiveTemplateByCode(clientId, req.EventType)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.TP_NOT_FOUND_001, "no active template for event type")
		return
	}

	recipients, err := repository.CheckIn.GetActiveCheckIns(clientId, branchId, now)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	recipients = alerting.FilterEligibleRecipients(recipients, now)

	event, err := createEvent(repository, ctx, req, template, len(recipients), now, "EM")
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}

	logTtl := time.Duration(setting.RetentionHours) * time.Hour
	outcome := repository.Dispatcher.DispatchAlert(event, recipients, template, logTtl)
	finalizeDispatch(repository, &event, outcome)

	if req.EventType == constant.EventAllClear {
		_ = repository.EmergencyEvent.CloseOpenEvents(clientId, branchId, now)
	}

	auditTrigger(ctx, repository, req.EventType, constant.ResultSuccess, event.EventNo)
	response.Ok(ctx, gin.H{
		"event":          event,
		"allFailed":      isAllFailed(event.ChannelSummary, event.RecipientCount),
		"channelSummary": event.ChannelSummary,
	})
}

func handleTestAlert(ctx *gin.Context, repository *domain.Repository, req request.TriggerAlert) {
	role := ctx.GetString("Role")
	if role != constant.SUPER && role != constant.ADMIN && role != constant.MANAGER {
		errcode.Abort(ctx, http.StatusForbidden, errcode.EM_FORBIDDEN_002, "test mode requires MANAGER or above")
		return
	}
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	now := time.Now()

	setting, err := repository.BranchSetting.GetSetting(clientId, branchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	if !verifyConfirmation(ctx, repository, setting, req) {
		return
	}
	template, err := repository.MessageTemplate.GetActiveTemplateByCode(clientId, constant.EventTest)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.TP_NOT_FOUND_001, "no active template for event type")
		return
	}
	testRecipients, err := repository.StaffPermission.GetTestRecipients(clientId, branchId)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}
	if len(testRecipients) == 0 {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_002, "no test recipients registered")
		return
	}

	event, err := createEvent(repository, ctx, req, template, len(testRecipients), now, "TS")
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return
	}

	logTtl := time.Duration(setting.RetentionHours) * time.Hour
	outcome := repository.Dispatcher.DispatchTest(event, testRecipients, template, logTtl)
	finalizeDispatch(repository, &event, outcome)

	repository.AuditLog.Record(entities.AuditLog{
		ClientId: clientId, BranchId: branchId,
		Actor: ctx.GetString("UserId"), Action: constant.ActionTestAlert,
		Detail: bson.M{"eventNo": event.EventNo, "recipientCount": event.RecipientCount},
		Result: constant.ResultSuccess,
	})
	response.Ok(ctx, gin.H{"event": event, "channelSummary": event.ChannelSummary})
}

func verifyConfirmation(ctx *gin.Context, repository *domain.Repository, setting entities.BranchSetting, req request.TriggerAlert) bool {
	if setting.ConfirmMethod == constant.ConfirmPin {
		if req.ConfirmMethod != constant.ConfirmPin {
			errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_002, "branch requires PIN confirmation")
			return false
		}
		return verifyPin(ctx, repository, setting, req.Pin)
	}
	if req.ConfirmMethod != constant.ConfirmHold3s {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_002, "invalid confirm method")
		return false
	}
	return true
}

func verifyPin(ctx *gin.Context, repository *domain.Repository, setting entities.BranchSetting, pin string) bool {
	if !setting.HasPin() {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.EM_BAD_REQUEST_002, "emergency PIN not configured")
		return false
	}
	lockKey := fmt.Sprintf("pinlock:%s:%s", setting.ClientId, setting.BranchId)
	attempts, _ := repository.RateLimit.Get(lockKey)
	if attempts >= constant.PinMaxAttempts {
		errcode.Abort(ctx, http.StatusLocked, errcode.EM_LOCKED_001, "PIN locked, try again later")
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(setting.PinHash), []byte(pin)); err != nil {
		count, _ := repository.RateLimit.Increment(lockKey, time.Duration(constant.PinLockMinutes)*time.Minute)
		if count >= constant.PinMaxAttempts {
			notifyPinLocked(repository, setting, ctx.GetString("UserId"))
		}
		errcode.Abort(ctx, http.StatusForbidden, errcode.EM_FORBIDDEN_002, "invalid PIN")
		return false
	}
	_ = repository.RateLimit.Reset(lockKey)
	return true
}

func notifyPinLocked(repository *domain.Repository, setting entities.BranchSetting, actor string) {
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: setting.ClientId, BranchId: setting.BranchId,
		Actor: actor, Action: constant.ActionChangePin,
		Detail: bson.M{"event": "PIN_LOCKED"},
		Result: constant.ResultFailed,
	})
}

func passCooldown(ctx *gin.Context, repository *domain.Repository, req request.TriggerAlert, cooldownSeconds int, now time.Time) bool {
	if alerting.IsCooldownExempt(req.EventType) {
		return true
	}
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	latest, err := repository.EmergencyEvent.GetLatestEvent(clientId, branchId, req.EventType)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.EM_INTERNAL_001, err.Error())
		return false
	}
	if latest == nil {
		return true
	}
	remaining := alerting.CooldownRemaining(&latest.SentAt, now, cooldownSeconds)
	if remaining <= 0 {
		return true
	}
	role := ctx.GetString("Role")
	canOverride := role == constant.SUPER || role == constant.ADMIN || role == constant.MANAGER
	if req.OverrideCooldown && canOverride {
		repository.AuditLog.Record(entities.AuditLog{
			ClientId: clientId, BranchId: branchId,
			Actor: ctx.GetString("UserId"), Action: constant.ActionCooldownOverride,
			Detail: bson.M{"eventType": req.EventType, "lastSentAt": latest.SentAt},
			Result: constant.ResultSuccess,
		})
		return true
	}
	errcode.Abort(ctx, http.StatusConflict, errcode.EM_CONFLICT_001,
		fmt.Sprintf("same event sent at %s, retry in %.0f seconds", latest.SentAt.Format(time.RFC3339), remaining.Seconds()))
	return false
}

func createEvent(repository *domain.Repository, ctx *gin.Context, req request.TriggerAlert, template entities.MessageTemplate, recipientCount int, now time.Time, prefix string) (entities.EmergencyEvent, error) {
	clientId := ctx.GetString("ClientId")
	sequence, err := repository.Counter.NextSequence(clientId, prefix, now)
	if err != nil {
		return entities.EmergencyEvent{}, err
	}
	event := entities.EmergencyEvent{
		EventNo:            alerting.FormatEventNo(prefix, now, sequence),
		ClientId:           clientId,
		BranchId:           ctx.GetString("BranchId"),
		EventType:          req.EventType,
		TemplateId:         template.Id,
		TriggeredBy:        ctx.GetString("UserId"),
		ConfirmedWith:      req.ConfirmMethod,
		CooldownOverridden: req.OverrideCooldown,
		RecipientCount:     recipientCount,
		Status:             constant.EventStatusOpen,
		SentAt:             now,
	}
	if req.EventType == constant.EventAllClear || req.EventType == constant.EventTest {
		event.Status = constant.EventStatusClosed
	}
	return repository.EmergencyEvent.CreateEvent(event)
}

func finalizeDispatch(repository *domain.Repository, event *entities.EmergencyEvent, outcome messaging.DispatchOutcome) {
	if err := repository.DeliveryLog.CreateMany(event.ClientId, outcome.Logs); err != nil {
		return
	}
	event.ChannelSummary = outcome.Summary
	event.ProviderReference = outcome.ProviderReference
	_ = repository.EmergencyEvent.UpdateChannelSummary(event.ClientId, event.Id, outcome.Summary, outcome.ProviderReference)
	for _, checkInId := range outcome.GoneSubscriptionIds {
		_ = repository.CheckIn.ClearPushSubscription(event.ClientId, checkInId)
	}
}

func isAllFailed(summary entities.ChannelSummary, recipientCount int) bool {
	if recipientCount == 0 {
		return false
	}
	return summary.Sms.Sent == 0 && summary.Push.Sent == 0 && summary.Line.Sent == 0
}

func auditTrigger(ctx *gin.Context, repository *domain.Repository, eventType string, result string, detail string) {
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: ctx.GetString("ClientId"),
		BranchId: ctx.GetString("BranchId"),
		Actor:    ctx.GetString("UserId"),
		Action:   constant.ActionTriggerAlert,
		Detail:   bson.M{"eventType": eventType, "info": detail},
		Result:   result,
	})
}
