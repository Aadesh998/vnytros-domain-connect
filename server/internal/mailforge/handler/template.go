package handler

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetTemplates(c *gin.Context) {
	lastID := parseUintQuery(c, "last_id", 0)
	limit := parseIntQuery(c, "limit", 10, 1, 100)

	response, appErr := services.GetTemplates(c.Request.Context(), lastID, limit)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func GetDraftTemplates(c *gin.Context) {
	lastID := parseUintQuery(c, "last_id", 0)
	limit := parseIntQuery(c, "limit", 10, 1, 100)

	response, appErr := services.GetDraftTemplates(c.Request.Context(), lastID, limit)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func CreateTemplate(c *gin.Context) {
	var req dto.TemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.CreateTemplate(c.Request.Context(), req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusCreated, response)
}

func GetTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	response, appErr := services.GetTemplate(c.Request.Context(), id)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func UpdateTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	var req dto.TemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apperror.BadRequest.WithMessage(err.Error()).SendError(c)
		return
	}

	response, appErr := services.UpdateTemplate(c.Request.Context(), id, req)
	if appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, response)
}

func DeleteTemplate(c *gin.Context) {
	id, ok := parseUintParam(c, "id")
	if !ok {
		apperror.BadRequest.SendError(c)
		return
	}

	if appErr := services.DeleteTemplate(c.Request.Context(), id); appErr != nil {
		appErr.SendError(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "Template deleted successfully"})
}
