package handler

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

const maxAudienceUpload = 16 << 20

func GetCampaigns(c *gin.Context) {
	lastID := parseUintQuery(c, "last_id", 0)
	limit := parseIntQuery(c, "limit", 10, 1, 100)

	response, appErr := services.GetCampaigns(c.Request.Context(), lastID, limit)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func GetDraftCampaigns(c *gin.Context) {
	lastID := parseUintQuery(c, "last_id", 0)
	limit := parseIntQuery(c, "limit", 10, 1, 100)

	response, appErr := services.GetDraftCampaigns(c.Request.Context(), lastID, limit)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func SendCampaign(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAudienceUpload)

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		apperror.BadRequest.WithMessage("An audience CSV is required in the 'file' field").SendError(c)
		return
	}
	defer file.Close()

	smtpConfigID := parseUintQuery(c, "smtp_config_id", 0)

	response, appErr := services.SendCampaign(c.Request.Context(), id, smtpConfigID, file)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusAccepted, response)
}

func CreateCampaign(c *gin.Context) {
	var req dto.CampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.CreateCampaign(c.Request.Context(), req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusCreated, response)
}

func GetCampaign(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	response, appErr := services.GetCampaign(c.Request.Context(), id)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func UpdateCampaign(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	var req dto.CampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.UpdateCampaign(c.Request.Context(), id, req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func DeleteCampaign(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	if appErr := services.DeleteCampaign(c.Request.Context(), id); appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Campaign deleted successfully"})
}
