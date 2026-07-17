package admin

import (
	"net/http"
	"os"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/data/entities"
	"alert/app/domain"
	"alert/app/domain/request"

	"github.com/gin-gonic/gin"
	qrcode "github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func validateTemplateTexts(req request.UpsertTemplate) (string, bool) {
	if !constant.IsValidEventType(req.Code) {
		return "invalid template code", false
	}
	for _, text := range []string{req.TextTh, req.TextEn} {
		if err := alerting.ValidateNoLink(text); err != nil {
			return err.Error(), false
		}
	}
	for _, override := range req.ChannelOverrides {
		for _, text := range []string{override.TextTh, override.TextEn} {
			if err := alerting.ValidateNoLink(text); err != nil {
				return err.Error(), false
			}
		}
	}
	return "", true
}

func templateFromRequest(req request.UpsertTemplate, clientId string, updatedBy string) entities.MessageTemplate {
	overrides := map[string]entities.ChannelText{}
	for channel, override := range req.ChannelOverrides {
		overrides[channel] = entities.ChannelText{TextTh: override.TextTh, TextEn: override.TextEn}
	}
	return entities.MessageTemplate{
		ClientId:         clientId,
		Code:             req.Code,
		TextTh:           req.TextTh,
		TextEn:           req.TextEn,
		ChannelOverrides: overrides,
		Active:           req.Active,
		UpdatedBy:        updatedBy,
	}
}

func handleCreateTemplate(ctx *gin.Context, repository *domain.Repository) {
	var req request.UpsertTemplate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_001, err.Error())
		return
	}
	if message, ok := validateTemplateTexts(req); !ok {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_002, message)
		return
	}
	clientId := ctx.GetString("ClientId")
	template, err := repository.MessageTemplate.CreateTemplate(templateFromRequest(req, clientId, ctx.GetString("UserId")))
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.TP_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionUpdateTemplate, bson.M{"templateId": template.Id.Hex(), "code": template.Code})
	response.Ok(ctx, gin.H{
		"template":    template,
		"smsSegments": gin.H{"th": alerting.SmsSegmentCount(template.TextTh), "en": alerting.SmsSegmentCount(template.TextEn)},
	})
}

func handleUpdateTemplate(ctx *gin.Context, repository *domain.Repository) {
	id, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_001, "invalid id")
		return
	}
	var req request.UpsertTemplate
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_001, err.Error())
		return
	}
	if message, ok := validateTemplateTexts(req); !ok {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_002, message)
		return
	}
	clientId := ctx.GetString("ClientId")
	existing, err := repository.MessageTemplate.GetTemplateById(clientId, id)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.TP_NOT_FOUND_001, "template not found")
		return
	}
	if !req.Active {
		activeCount, countErr := repository.MessageTemplate.CountActiveByCode(clientId, existing.Code, &id)
		if countErr == nil && activeCount == 0 {
			errcode.Abort(ctx, http.StatusBadRequest, errcode.TP_BAD_REQUEST_002, "every event type must keep at least one active template")
			return
		}
	}
	template := templateFromRequest(req, clientId, ctx.GetString("UserId"))
	if err := repository.MessageTemplate.UpdateTemplate(clientId, id, template); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.TP_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionUpdateTemplate, bson.M{"templateId": id.Hex(), "code": existing.Code})
	response.Ok(ctx, gin.H{
		"smsSegments": gin.H{"th": alerting.SmsSegmentCount(req.TextTh), "en": alerting.SmsSegmentCount(req.TextEn)},
	})
}

func handleUpdateSetting(ctx *gin.Context, repository *domain.Repository) {
	var req request.UpdateBranchSetting
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, err.Error())
		return
	}
	if req.RetentionHours < constant.MinRetentionHours || req.RetentionHours > constant.MaxRetentionHours {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "retentionHours must be between 6 and 24")
		return
	}
	if req.CooldownSeconds < constant.MinCooldownSeconds || req.CooldownSeconds > constant.MaxCooldownSeconds {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "cooldownSeconds must be between 60 and 180")
		return
	}
	if req.ConfirmMethod != constant.ConfirmHold3s && req.ConfirmMethod != constant.ConfirmPin {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "invalid confirm method")
		return
	}
	setting := entities.BranchSetting{
		ClientId:           ctx.GetString("ClientId"),
		BranchId:           ctx.GetString("BranchId"),
		ShopName:           req.ShopName,
		RetentionHours:     req.RetentionHours,
		CooldownSeconds:    req.CooldownSeconds,
		ConfirmMethod:      req.ConfirmMethod,
		SmsCreditThreshold: req.SmsCreditThreshold,
		ContactChannel:     req.ContactChannel,
		UpdatedBy:          ctx.GetString("UserId"),
	}
	if err := repository.BranchSetting.UpsertSetting(setting); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionUpdateSetting, bson.M{"retentionHours": req.RetentionHours, "cooldownSeconds": req.CooldownSeconds})
	response.Ok(ctx, setting)
}

func handleSetPin(ctx *gin.Context, repository *domain.Repository) {
	var req request.SetPin
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, err.Error())
		return
	}
	if req.ConfirmMethod != constant.ConfirmHold3s && req.ConfirmMethod != constant.ConfirmPin {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "invalid confirm method")
		return
	}
	pinHash, err := bcrypt.GenerateFromPassword([]byte(req.Pin), bcrypt.DefaultCost)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	if err := repository.BranchSetting.SetPinHash(clientId, branchId, string(pinHash), req.ConfirmMethod, ctx.GetString("UserId")); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionChangePin, bson.M{"confirmMethod": req.ConfirmMethod})
	response.Ok(ctx, gin.H{"updated": true})
}

func handleUpsertPermission(ctx *gin.Context, repository *domain.Repository) {
	var req request.UpsertPermission
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, err.Error())
		return
	}
	for _, eventType := range req.AllowedEventTypes {
		if !constant.IsValidEventType(eventType) {
			errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "invalid event type: "+eventType)
			return
		}
	}
	phone := ""
	if req.Phone != "" {
		normalized, err := alerting.NormalizeThaiPhone(req.Phone)
		if err != nil {
			errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_002, "invalid phone number")
			return
		}
		phone = normalized
	}
	permission := entities.StaffPermission{
		ClientId:          ctx.GetString("ClientId"),
		UserId:            req.UserId,
		BranchId:          req.BranchId,
		Phone:             phone,
		AllowedEventTypes: req.AllowedEventTypes,
		IsTestRecipient:   req.IsTestRecipient,
		Active:            req.Active,
		UpdatedBy:         ctx.GetString("UserId"),
	}
	if err := repository.StaffPermission.UpsertPermission(permission); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionUpdatePermission, bson.M{"targetUserId": req.UserId})
	response.Ok(ctx, gin.H{"updated": true})
}

func handleCreateQr(ctx *gin.Context, repository *domain.Repository) {
	var req request.CreateQr
	if err := ctx.ShouldBindJSON(&req); err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, err.Error())
		return
	}
	randomToken, err := alerting.GenerateSessionToken()
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	clientId := ctx.GetString("ClientId")
	token := alerting.ComposeTenantRef(clientId, randomToken)
	qrToken, err := repository.QrToken.CreateQrToken(entities.QrToken{
		ClientId:  clientId,
		BranchId:  req.BranchId,
		TableNo:   req.TableNo,
		Token:     token,
		CreatedBy: ctx.GetString("UserId"),
	})
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionManageQr, bson.M{"qrId": qrToken.Id.Hex(), "action": "CREATE"})
	response.Ok(ctx, gin.H{"qr": qrToken, "checkInUrl": checkInUrl(token)})
}

func handleRevokeQr(ctx *gin.Context, repository *domain.Repository) {
	id, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, "invalid id")
		return
	}
	if err := repository.QrToken.Revoke(ctx.GetString("ClientId"), id, time.Now()); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	auditAdmin(ctx, repository, constant.ActionManageQr, bson.M{"qrId": id.Hex(), "action": "REVOKE"})
	response.Ok(ctx, gin.H{"revoked": true})
}

func handleQrImage(ctx *gin.Context, repository *domain.Repository) {
	id, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.AD_BAD_REQUEST_001, "invalid id")
		return
	}
	qrToken, err := repository.QrToken.GetQrTokenById(ctx.GetString("ClientId"), id)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.AD_NOT_FOUND_001, "QR not found")
		return
	}
	png, err := qrcode.Encode(checkInUrl(qrToken.Token), qrcode.Medium, 512)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.AD_INTERNAL_001, err.Error())
		return
	}
	ctx.Data(http.StatusOK, "image/png", png)
}

func handleSmsCredit(ctx *gin.Context, repository *domain.Repository) {
	if repository.SmsBalance == nil {
		response.Ok(ctx, gin.H{"available": false, "credit": 0})
		return
	}
	credit, err := repository.SmsBalance.Balance(repository.ProviderConfigFor(ctx.GetString("ClientId")))
	if err != nil {
		response.Ok(ctx, gin.H{"available": false, "credit": 0, "reason": err.Error()})
		return
	}
	setting, _ := repository.BranchSetting.GetSetting(ctx.GetString("ClientId"), ctx.GetString("BranchId"))
	response.Ok(ctx, gin.H{
		"available": true,
		"credit":    credit,
		"threshold": setting.SmsCreditThreshold,
		"low":       setting.SmsCreditThreshold > 0 && credit < int64(setting.SmsCreditThreshold),
	})
}

func checkInUrl(token string) string {
	base := os.Getenv("CHECKIN_BASE_URL")
	if base == "" {
		base = "https://alert.devper.app/checkin"
	}
	return base + "?token=" + token
}

func auditAdmin(ctx *gin.Context, repository *domain.Repository, action string, detail bson.M) {
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: ctx.GetString("ClientId"),
		BranchId: ctx.GetString("BranchId"),
		Actor:    ctx.GetString("UserId"),
		Action:   action,
		Detail:   detail,
		Result:   constant.ResultSuccess,
	})
}
