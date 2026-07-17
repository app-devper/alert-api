package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/domain"

	"github.com/gin-gonic/gin"
)

type smsDeliveryReport struct {
	Reference string `json:"reference"`
	Status    string `json:"status"`
}

func ApplyWebhookAPI(route *gin.RouterGroup, repository *domain.Repository) {
	r := route.Group("webhook")

	r.POST("/sms", func(ctx *gin.Context) {
		handleSmsWebhook(ctx, repository)
	})

	r.POST("/line", func(ctx *gin.Context) {
		handleLineWebhook(ctx, repository)
	})
}

func handleSmsWebhook(ctx *gin.Context, repository *domain.Repository) {
	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.WH_BAD_REQUEST_001, "unreadable body")
		return
	}
	secret := repository.ProviderConfigFor(webhookClientId(ctx, repository)).SmsWebhookSecret
	if !verifySignature(secret, ctx.GetHeader("X-Webhook-Signature"), body) {
		errcode.Abort(ctx, http.StatusUnauthorized, errcode.WH_UNAUTHORIZED_001, "invalid signature")
		return
	}
	var reports []smsDeliveryReport
	if err := json.Unmarshal(body, &reports); err != nil {
		var single smsDeliveryReport
		if err := json.Unmarshal(body, &single); err != nil {
			errcode.Abort(ctx, http.StatusBadRequest, errcode.WH_BAD_REQUEST_001, "invalid payload")
			return
		}
		reports = []smsDeliveryReport{single}
	}
	now := time.Now()
	updated := 0
	tenants := candidateTenants(ctx.Query("clientId"), repository)
	for _, report := range reports {
		status := mapProviderStatus(report.Status)
		if report.Reference == "" || status == "" {
			continue
		}
		for _, clientId := range tenants {
			if _, err := repository.DeliveryLog.UpdateStatusByProviderReference(clientId, report.Reference, status, report.Status, now); err == nil {
				updated++
				break
			}
		}
	}
	response.Ok(ctx, gin.H{"updated": updated})
}

func webhookClientId(ctx *gin.Context, repository *domain.Repository) string {
	if clientId := ctx.Query("clientId"); clientId != "" {
		return clientId
	}
	known := repository.Tenants.KnownClients()
	if len(known) == 1 {
		return known[0]
	}
	return ""
}

func candidateTenants(clientId string, repository *domain.Repository) []string {
	if clientId != "" {
		return []string{clientId}
	}
	return repository.Tenants.KnownClients()
}

func verifySignature(secret string, signature string, body []byte) bool {
	if secret == "" {
		return true
	}
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func mapProviderStatus(providerStatus string) string {
	switch providerStatus {
	case "DELIVERED", "DELIVRD", "delivered":
		return constant.DeliveryDelivered
	case "FAILED", "UNDELIV", "EXPIRED", "REJECTED", "failed":
		return constant.DeliveryFailed
	case "SENT", "ACCEPTED", "sent":
		return constant.DeliverySent
	}
	return ""
}
