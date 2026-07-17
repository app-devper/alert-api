package dashboard

import (
	"net/http"
	"time"

	"alert/app/core/alerting"
	"alert/app/core/constant"
	"alert/app/core/errcode"
	"alert/app/core/response"
	"alert/app/data/entities"
	"alert/app/data/repositories"
	"alert/app/domain"
	"alert/middlewares"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ApplyDashboardAPI(route *gin.RouterGroup, repository *domain.Repository) {
	r := route.Group("dashboard",
		middlewares.RequireAuthenticated(),
		middlewares.RequireSession(repository.Session),
		middlewares.RequireBranch(repository.StaffPermission),
		middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER, constant.STAFF),
	)

	r.GET("/summary", func(ctx *gin.Context) {
		handleSummary(ctx, repository)
	})

	r.GET("/check-ins", func(ctx *gin.Context) {
		handleCheckInList(ctx, repository)
	})

	r.POST("/check-ins/:id/checkout", func(ctx *gin.Context) {
		handleStaffCheckout(ctx, repository)
	})

	manager := r.Group("", middlewares.RequireAuthorization(constant.SUPER, constant.ADMIN, constant.MANAGER))

	manager.GET("/events", func(ctx *gin.Context) {
		handleEventHistory(ctx, repository)
	})

	manager.GET("/events/:id", func(ctx *gin.Context) {
		handleEventDetail(ctx, repository)
	})
}

func handleSummary(ctx *gin.Context, repository *domain.Repository) {
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	now := time.Now()
	checkIns, err := repository.CheckIn.GetActiveCheckIns(clientId, branchId, now)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.DA_INTERNAL_001, err.Error())
		return
	}
	active := alerting.FilterEligibleRecipients(checkIns, now)
	totalPeople, pushCount := 0, 0
	for _, checkIn := range active {
		totalPeople += checkIn.GroupSize
		if checkIn.HasPush() {
			pushCount++
		}
	}
	response.Ok(ctx, gin.H{
		"activeCheckIns": len(active),
		"totalPeople":    totalPeople,
		"pushEnabled":    pushCount,
		"lineEnabled":    len(active),
		"branchId":       branchId,
		"asOf":           now,
	})
}

type checkInListItem struct {
	Id          string    `json:"id"`
	CheckInNo   string    `json:"checkInNo"`
	PhoneMasked string    `json:"phoneMasked"`
	TableNo     string    `json:"tableNo"`
	GroupSize   int       `json:"groupSize"`
	CheckedInAt time.Time `json:"checkedInAt"`
	HasPush     bool      `json:"hasPush"`
}

func handleCheckInList(ctx *gin.Context, repository *domain.Repository) {
	clientId := ctx.GetString("ClientId")
	branchId := ctx.GetString("BranchId")
	search := ctx.Query("search")
	now := time.Now()
	checkIns, err := repository.CheckIn.GetActiveCheckIns(clientId, branchId, now)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.DA_INTERNAL_001, err.Error())
		return
	}
	items := make([]checkInListItem, 0, len(checkIns))
	for _, checkIn := range alerting.FilterEligibleRecipients(checkIns, now) {
		if !matchesSearch(checkIn, search) {
			continue
		}
		items = append(items, checkInListItem{
			Id:          checkIn.Id.Hex(),
			CheckInNo:   checkIn.CheckInNo,
			PhoneMasked: alerting.MaskPhoneDisplay(checkIn.Phone),
			TableNo:     checkIn.TableNo,
			GroupSize:   checkIn.GroupSize,
			CheckedInAt: checkIn.CheckedInAt,
			HasPush:     checkIn.HasPush(),
		})
	}
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: clientId, BranchId: branchId,
		Actor: ctx.GetString("UserId"), Action: constant.ActionViewCheckinList,
		Detail: bson.M{"count": len(items), "search": search != ""},
		Result: constant.ResultSuccess,
	})
	response.Ok(ctx, items)
}

func matchesSearch(checkIn entities.CheckIn, search string) bool {
	if search == "" {
		return true
	}
	return checkIn.TableNo == search || alerting.PhoneLast4(checkIn.Phone) == search
}

func handleStaffCheckout(ctx *gin.Context, repository *domain.Repository) {
	id, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.DA_BAD_REQUEST_001, "invalid id")
		return
	}
	userId := ctx.GetString("UserId")
	if err := repository.CheckIn.Checkout(ctx.GetString("ClientId"), id, time.Now(), "STAFF:"+userId); err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.DA_INTERNAL_001, err.Error())
		return
	}
	repository.AuditLog.Record(entities.AuditLog{
		ClientId: ctx.GetString("ClientId"), BranchId: ctx.GetString("BranchId"),
		Actor: userId, Action: constant.ActionCheckoutCustomer,
		Detail: bson.M{"checkInId": id.Hex()},
		Result: constant.ResultSuccess,
	})
	response.Ok(ctx, gin.H{"status": "CHECKED_OUT"})
}

func handleEventHistory(ctx *gin.Context, repository *domain.Repository) {
	query := repositories.EventQuery{
		ClientId:  ctx.GetString("ClientId"),
		BranchId:  ctx.GetString("BranchId"),
		EventType: ctx.Query("eventType"),
		Page:      parseInt(ctx.Query("page"), 1),
		Limit:     parseInt(ctx.Query("limit"), 20),
	}
	if from, err := time.Parse(time.RFC3339, ctx.Query("from")); err == nil {
		query.From = &from
	}
	if to, err := time.Parse(time.RFC3339, ctx.Query("to")); err == nil {
		query.To = &to
	}
	events, total, err := repository.EmergencyEvent.QueryEvents(query)
	if err != nil {
		errcode.Abort(ctx, http.StatusInternalServerError, errcode.DA_INTERNAL_001, err.Error())
		return
	}
	response.OkWithMeta(ctx, events, response.Meta{Total: total, Page: query.Page, Limit: query.Limit})
}

func handleEventDetail(ctx *gin.Context, repository *domain.Repository) {
	id, err := primitive.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		errcode.Abort(ctx, http.StatusBadRequest, errcode.DA_BAD_REQUEST_001, "invalid id")
		return
	}
	clientId := ctx.GetString("ClientId")
	event, err := repository.EmergencyEvent.GetEventById(clientId, id)
	if err != nil {
		errcode.Abort(ctx, http.StatusNotFound, errcode.EM_NOT_FOUND_001, "event not found")
		return
	}
	summary, err := repository.DeliveryLog.SummarizeByEventId(clientId, id)
	if err == nil {
		event.ChannelSummary = summary
	}
	failed, err := repository.DeliveryLog.GetFailedByEventId(clientId, id)
	if err != nil {
		failed = []entities.DeliveryLog{}
	}
	response.Ok(ctx, gin.H{
		"event":            event,
		"failedDeliveries": failed,
	})
}

func parseInt(value string, fallback int64) int64 {
	if value == "" {
		return fallback
	}
	var parsed int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return fallback
		}
		parsed = parsed*10 + int64(r-'0')
	}
	if parsed == 0 {
		return fallback
	}
	return parsed
}
