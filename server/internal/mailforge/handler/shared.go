package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func parseUintParam(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || v == 0 {
		return 0, false
	}
	return uint(v), true
}

func parseUintQuery(c *gin.Context, name string, def uint) uint {
	raw := c.Query(name)
	if raw == "" {
		return def
	}
	v, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		return def
	}
	return uint(v)
}

func parseIntQuery(c *gin.Context, name string, def, minVal, maxVal int) int {
	raw := c.Query(name)
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	if v < minVal {
		return minVal
	}
	if v > maxVal {
		return maxVal
	}
	return v
}
