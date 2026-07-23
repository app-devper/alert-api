package app

import (
	"net/http"

	"alert/app/core/config"
	"alert/app/core/response"
	"alert/app/domain"
	"alert/app/featues/admin"
	"alert/app/featues/checkin"
	"alert/app/featues/dashboard"
	"alert/app/featues/emergency"
	"alert/app/featues/webhook"
	"alert/db"
	"alert/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Routes struct{}

func (app Routes) StartGin() {
	cfg := config.MustLoad()

	r := gin.New()

	err := r.SetTrustedProxies(nil)
	if err != nil {
		logrus.Error(err)
	}

	r.Use(gin.Logger())
	r.Use(middlewares.NewRecovery())
	r.Use(middlewares.NewCors([]string{"*"}))

	resource, err := db.InitResource(cfg, domain.TemplateSeeder())
	if err != nil {
		logrus.Fatal("failed to init database: ", err)
	}
	defer resource.Close()

	repository := domain.InitRepository(resource, cfg)

	RegisterRoutes(r, repository)

	err = r.Run(":" + cfg.Port)
	if err != nil {
		logrus.Error(err)
	}
}

func RegisterRoutes(r *gin.Engine, repository *domain.Repository) {
	r.GET("/health", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	publicRoute := r.Group("/api/alert/v1")
	publicRoute.GET("/health", func(ctx *gin.Context) {
		response.Ok(ctx, gin.H{"status": "ok"})
	})

	checkin.ApplyCheckInAPI(publicRoute, repository)
	emergency.ApplyEmergencyAPI(publicRoute, repository)
	dashboard.ApplyDashboardAPI(publicRoute, repository)
	admin.ApplyAdminAPI(publicRoute, repository)
	webhook.ApplyWebhookAPI(publicRoute, repository)

	r.NoRoute(middlewares.NoRoute())
}
