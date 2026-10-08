package handler

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetBranding(c *gin.Context) {
	response, appErr := services.GetBranding(c.Request.Context())
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func UpdateBranding(c *gin.Context) {
	var req dto.BrandingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.UpdateBranding(c.Request.Context(), req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}
