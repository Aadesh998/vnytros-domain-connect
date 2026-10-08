package handler

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ListSmtpConfigs(c *gin.Context) {
	response, appErr := services.ListSmtpConfigs(c.Request.Context())
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func GetSmtpConfig(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	response, appErr := services.GetSmtpConfig(c.Request.Context(), id)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func CreateSmtpConfig(c *gin.Context) {
	var req dto.SmtpConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.CreateSmtpConfig(c.Request.Context(), req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusCreated, response)
}

func UpdateSmtpConfig(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	var req dto.SmtpConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.UpdateSmtpConfig(c.Request.Context(), id, req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func SetDefaultSmtpConfig(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	response, appErr := services.SetDefaultSmtpConfig(c.Request.Context(), id)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func DeleteSmtpConfig(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	if appErr := services.DeleteSmtpConfig(c.Request.Context(), id); appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Sender deleted"})
}

func SeedSmtpConfig(c *gin.Context) {
	response, appErr := services.SeedFallbackSmtpConfig(c.Request.Context())
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}
