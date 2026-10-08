package mail

import (
	"crypto/tls"
	"fmt"
	"html"
	"net/url"
	"strings"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/models"

	gomail "gopkg.in/mail.v2"
)

const (
	EncryptionStartTLS = "starttls"
	EncryptionSSL      = "ssl"
	EncryptionNone     = "none"
)

func NewCampaignDialer(cfg models.MailSmtpConfig) *gomail.Dialer {
	username := cfg.Username
	if username == "" {
		username = cfg.FromEmail
	}

	port := cfg.Port
	if port <= 0 {
		port = 587
	}

	d := gomail.NewDialer(cfg.Host, port, username, cfg.Password)
	d.TLSConfig = &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}

	switch strings.ToLower(cfg.Encryption) {
	case EncryptionSSL:
		d.SSL = true
	case EncryptionNone:
		d.SSL = false
		d.StartTLSPolicy = gomail.OpportunisticStartTLS
	default: // starttls
		d.SSL = false
		d.StartTLSPolicy = gomail.MandatoryStartTLS
	}

	return d
}

func NewCampaignMessage(cfg models.MailSmtpConfig, job jobs.CampaignSendJob, templateBody, recipient string) *gomail.Message {
	m := gomail.NewMessage()

	if cfg.FromName != "" {
		m.SetAddressHeader("From", cfg.FromEmail, cfg.FromName)
	} else {
		m.SetHeader("From", cfg.FromEmail)
	}
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", job.Subject)

	body := templateBody +
		WatermarkFooter(job.Watermark) +
		TrackingPixel(job.TrackBaseURL, job.CampaignID, job.TemplateID, recipient)

	m.SetBody("text/html", body)
	return m
}

func WatermarkFooter(wm jobs.Watermark) string {
	if !wm.Show || wm.ImageURL == "" {
		return ""
	}

	link := wm.LinkURL
	label := wm.Label
	if label == "" {
		label = "Sent with Vnytros"
	}

	if link == "" {
		return fmt.Sprintf(`
<div style="text-align:center;margin-top:24px">
  <hr style="border:0;border-top:1px solid #eee;margin:20px 0">
  <img src="%s" alt="%s" style="width:110px;height:auto;display:block;margin:0 auto">
  <p style="font-size:12px;color:#777;margin-top:10px">%s</p>
</div>`,
			html.EscapeString(wm.ImageURL),
			html.EscapeString(label),
			html.EscapeString(label),
		)
	}

	return fmt.Sprintf(`
<div style="text-align:center;margin-top:24px">
  <hr style="border:0;border-top:1px solid #eee;margin:20px 0">
  <a href="%s" target="_blank" rel="noopener">
    <img src="%s" alt="%s" style="width:110px;height:auto;display:block;margin:0 auto">
  </a>
  <p style="font-size:12px;color:#777;margin-top:10px">
    <a href="%s" style="color:#007bff;text-decoration:none">%s</a>
  </p>
</div>`,
		html.EscapeString(link),
		html.EscapeString(wm.ImageURL),
		html.EscapeString(label),
		html.EscapeString(link),
		html.EscapeString(label),
	)
}

func TrackingPixel(baseURL string, campaignID, templateID uint, recipient string) string {
	if baseURL == "" {
		return ""
	}

	q := url.Values{}
	q.Set("cid", fmt.Sprintf("%d", campaignID))
	q.Set("tid", fmt.Sprintf("%d", templateID))
	q.Set("email", recipient)

	src := fmt.Sprintf("%s/api/track?%s", strings.TrimRight(baseURL, "/"), q.Encode())
	return fmt.Sprintf(`<img src="%s" width="1" height="1" alt="" style="display:none!important" />`, html.EscapeString(src))
}
