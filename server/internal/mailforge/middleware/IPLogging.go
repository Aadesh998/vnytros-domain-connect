package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
)

func IPLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		log.Printf("[INFO]: Request - IP: %s, Method: %s, Path: %s", ip, c.Request.Method, c.Request.URL.Path)
		c.Next()
	}
}
