package handler

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetAccountAnalytics(c *gin.Context) {
	days := parseIntQuery(c, "days", 30, 1, 365)

	response, appErr := services.GetAccountAnalytics(c.Request.Context(), days, c.Query("interval"))
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func GetCampaignAnalytics(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	days := parseIntQuery(c, "days", 30, 1, 365)

	response, appErr := services.GetCampaignAnalytics(c.Request.Context(), id, days, c.Query("interval"))
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func ListCampaignRecipients(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	offset := parseIntQuery(c, "offset", 0, 0, 1_000_000)
	limit := parseIntQuery(c, "limit", 50, 1, 200)

	response, appErr := services.ListCampaignRecipients(c.Request.Context(), id, c.Query("status"), offset, limit)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}
