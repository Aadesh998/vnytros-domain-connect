package mail

import (
	"domain-connect-backend/internal/config"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	gomail "gopkg.in/mail.v2"
)

func buildFinalBody(body string) string {
	lower := strings.ToLower(body)
	if strings.Contains(lower, "<!doctype") || strings.Contains(lower, "<html") {
		return body
	}
	frontendURL := config.AppConfig.FrontendURL
	footer := fmt.Sprintf(`
	<br><hr>
	<p style="font-size: 14px; color: #777;">
	Visit our website: <a href="%s">%s</a>
	</p>
	`, frontendURL, frontendURL)
	return body + footer
}

func SendMail(to, subject, body string) {
	SendMailWithEmbeddedImage(to, subject, body, nil)
}

func SendMailWithEmbeddedImage(to, subject, body string, imageBytes []byte) {
	message := gomail.NewMessage()
	from := config.AppConfig.SMTPFromEmail
	password := config.AppConfig.SMTPPassword
	host := config.AppConfig.SMTPHost
	portStr := config.AppConfig.SMTPPort

	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = 587
	}

	finalBody := buildFinalBody(body)

	message.SetHeader("From", from)
	message.SetHeader("To", to)
	message.SetHeader("Subject", subject)

	if len(imageBytes) > 0 {
		message.Embed("preview.png", gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(imageBytes)
			return err
		}))
	}

	message.SetBody("text/html", finalBody)

	d := gomail.NewDialer(host, port, from, password)

	if err := d.DialAndSend(message); err != nil {
		log.Printf("Failed to send email to %s: %v", to, err)
	} else {
		log.Printf("Email sent successfully to %s", to)
	}
}
