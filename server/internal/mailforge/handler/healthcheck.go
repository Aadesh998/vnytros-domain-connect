package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
)

type HealthCheckResponse struct {
	Status  string    `json:"status"`
	Time    time.Time `json:"time"`
	Message string    `json:"message"`
}

func HealthCheck(c *gin.Context) {

	ctx := c.Request.Context()
	tracer := otel.Tracer("healthcheck")
	_, span := tracer.Start(ctx, "healthcheck")
	defer span.End()

	HealthCheckResponse := HealthCheckResponse{
		Status:  "ok",
		Time:    time.Now(),
		Message: "Server is healthy",
	}
	c.JSON(200, HealthCheckResponse)
}
