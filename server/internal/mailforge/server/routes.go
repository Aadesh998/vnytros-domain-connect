package server

import (
	"domain-connect-backend/internal/mailforge/handler"

	"github.com/gin-gonic/gin"
)

func publicRoutes(r *gin.Engine) {
	r.GET("/api/health", handler.HealthCheck)
	r.GET("/api/track", handler.TrackOpen)
}

func routesTemplate(t *gin.RouterGroup) {
	t.GET("", handler.GetTemplates)
	t.GET("/", handler.GetTemplates)
	t.GET("/draft", handler.GetDraftTemplates)
	t.POST("", handler.CreateTemplate)
	t.POST("/", handler.CreateTemplate)
	t.GET("/:id", handler.GetTemplate)
	t.PUT("/:id", handler.UpdateTemplate)
	t.DELETE("/:id", handler.DeleteTemplate)
}

func routesCampaign(c *gin.RouterGroup) {
	c.GET("", handler.GetCampaigns)
	c.GET("/", handler.GetCampaigns)
	c.GET("/draft", handler.GetDraftCampaigns)
	c.POST("", handler.CreateCampaign)
	c.POST("/", handler.CreateCampaign)
	c.POST("/:id/send", handler.SendCampaign)
	c.GET("/:id", handler.GetCampaign)
	c.PUT("/:id", handler.UpdateCampaign)
	c.DELETE("/:id", handler.DeleteCampaign)

	c.GET("/:id/analytics", handler.GetCampaignAnalytics)
	c.GET("/:id/recipients", handler.ListCampaignRecipients)
}

func routesSettings(s *gin.RouterGroup) {
	s.GET("/smtp", handler.ListSmtpConfigs)
	s.POST("/smtp", handler.CreateSmtpConfig)
	s.POST("/smtp/seed", handler.SeedSmtpConfig)
	s.GET("/smtp/:id", handler.GetSmtpConfig)
	s.PUT("/smtp/:id", handler.UpdateSmtpConfig)
	s.POST("/smtp/:id/default", handler.SetDefaultSmtpConfig)
	s.DELETE("/smtp/:id", handler.DeleteSmtpConfig)
	s.GET("/branding", handler.GetBranding)
	s.PUT("/branding", handler.UpdateBranding)
}

func routesAnalytics(a *gin.RouterGroup) {
	a.GET("/overview", handler.GetAccountAnalytics)
}
