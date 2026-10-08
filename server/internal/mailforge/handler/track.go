package handler

import (
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mailforge/services"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var pixel = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
	0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54, 0x08, 0xD7, 0x63, 0x60, 0x00, 0x02, 0x00,
	0x00, 0x05, 0x00, 0x01, 0x0D, 0x26, 0xE5, 0x2E, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
	0xAE, 0x42, 0x60, 0x82,
}

func TrackOpen(c *gin.Context) {
	defer servePixel(c)

	cid, err := strconv.ParseUint(c.Query("cid"), 10, 32)
	if err != nil || cid == 0 {
		return
	}
	tid, _ := strconv.ParseUint(c.Query("tid"), 10, 32)

	recipient := strings.ToLower(strings.TrimSpace(c.Query("email")))
	if recipient == "" {
		return
	}

	job := jobs.CampaignTrackJob{
		CampaignID: uint(cid),
		TemplateID: uint(tid),
		Recipient:  recipient,
		EventType:  "open",
		UserAgent:  c.Request.UserAgent(),
		IPAddress:  c.ClientIP(),
	}

	if err := services.RecordOpen(c.Request.Context(), job); err != nil {
		log.Printf("track: failed to enqueue open for campaign %d: %v", cid, err)
	}
}

func servePixel(c *gin.Context) {
	c.Header("Content-Type", "image/png")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Data(http.StatusOK, "image/png", pixel)
}
