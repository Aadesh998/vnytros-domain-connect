package mail

import (
	"strings"
	"testing"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/models"
)

func TestWatermarkFooterHidden(t *testing.T) {
	t.Parallel()

	if got := WatermarkFooter(jobs.Watermark{Show: false, ImageURL: "x"}); got != "" {
		t.Errorf("footer with Show=false = %q, want empty", got)
	}
	if got := WatermarkFooter(jobs.Watermark{Show: true, ImageURL: ""}); got != "" {
		t.Errorf("footer with no image = %q, want empty", got)
	}
}

func TestWatermarkFooterEscapesUntrustedValues(t *testing.T) {
	t.Parallel()

	got := WatermarkFooter(jobs.Watermark{
		Show:     true,
		ImageURL: `https://x/a.png"><script>alert(1)</script>`,
		LinkURL:  `https://x/?a=1&b=2`,
		Label:    `Ada & "Co"`,
	})

	if strings.Contains(got, "<script>") {
		t.Errorf("footer contains unescaped script tag: %s", got)
	}
	if !strings.Contains(got, "&amp;") {
		t.Errorf("footer did not escape ampersand: %s", got)
	}
	if !strings.Contains(got, "&#34;") {
		t.Errorf("footer did not escape double quote: %s", got)
	}
}

func TestTrackingPixel(t *testing.T) {
	t.Parallel()

	if got := TrackingPixel("", 1, 2, "a@b.com"); got != "" {
		t.Errorf("pixel with no base URL = %q, want empty", got)
	}

	got := TrackingPixel("https://mail.example.com/", 7, 9, "a+tag@b.com")

	if strings.Contains(got, "com//api") {
		t.Errorf("pixel has doubled slash: %s", got)
	}
	for _, want := range []string{"cid=7", "tid=9", "a%2Btag%40b.com", `width="1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("pixel missing %q: %s", want, got)
		}
	}
}

func TestNewCampaignDialerEncryption(t *testing.T) {
	t.Parallel()

	ssl := NewCampaignDialer(models.MailSmtpConfig{Host: "h", Port: 465, Encryption: EncryptionSSL})
	if !ssl.SSL {
		t.Error("ssl encryption did not enable implicit TLS")
	}

	starttls := NewCampaignDialer(models.MailSmtpConfig{Host: "h", Port: 587, Encryption: EncryptionStartTLS})
	if starttls.SSL {
		t.Error("starttls must not enable implicit TLS")
	}

	if d := NewCampaignDialer(models.MailSmtpConfig{Host: "h", Port: 0}); d.Port != 587 {
		t.Errorf("default port = %d, want 587", d.Port)
	}

	d := NewCampaignDialer(models.MailSmtpConfig{Host: "h", Port: 587, FromEmail: "a@b.com"})
	if d.Username != "a@b.com" {
		t.Errorf("username = %q, want the from address", d.Username)
	}
}

func TestNewCampaignMessageIncludesFooterAndPixel(t *testing.T) {
	t.Parallel()

	job := jobs.CampaignSendJob{
		CampaignID:   3,
		TemplateID:   4,
		Subject:      "Hello",
		TrackBaseURL: "https://mail.example.com",
		Watermark:    jobs.Watermark{Show: true, ImageURL: "https://x/logo.svg", Label: "Sent"},
	}
	cfg := models.MailSmtpConfig{FromEmail: "from@x.com", Host: "h", Port: 587}

	msg := NewCampaignMessage(cfg, job, "<p>Body</p>", "to@y.com")

	var sb strings.Builder
	if _, err := msg.WriteTo(&sb); err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	out := strings.ReplaceAll(sb.String(), "=\r\n", "")

	for _, want := range []string{"Body", "logo.svg", "cid=3"} {
		if !strings.Contains(out, want) {
			t.Errorf("message missing %q", want)
		}
	}
}
