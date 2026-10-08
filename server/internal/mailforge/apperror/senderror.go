package apperror

import "github.com/gin-gonic/gin"

type AppError struct {
	Message    string
	Code       string
	HTTPStatus int
}

func (e AppError) Error() string { return e.Message }

func (e AppError) WithMessage(message string) *AppError {
	clone := e
	clone.Message = message
	return &clone
}

func (e AppError) SendError(c *gin.Context) {
	c.JSON(e.HTTPStatus, gin.H{
		"error":   e.Message,
		"code":    e.Code,
		"message": e.Message,
	})
}
