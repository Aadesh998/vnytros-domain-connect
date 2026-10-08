package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
)

func ErrorHandlingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Internal Server Error: %s", err)
				c.JSON(500, gin.H{
					"error":  "internal server error",
					"detail": err,
				})
				c.Abort()
			}
		}()

		c.Next()
	}
}
