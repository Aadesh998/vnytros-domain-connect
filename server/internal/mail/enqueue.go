package mail

import (
	"context"
	"log"

	"domain-connect-backend/internal/queue"
)

func Enqueue(to, subject, template string, vars map[string]string) {
	if err := queue.EnqueueEmail(context.Background(), to, subject, template, vars); err != nil {
		log.Printf("mail.Enqueue: %v — falling back to sync send", err)
		sendSync(to, subject, template, vars)
	}
}

func sendSync(to, subject, template string, vars map[string]string) {
	body := RenderTemplate(template, vars)
	if body == "" {
		log.Printf("mail.Enqueue: template %q rendered empty, skipping", template)
		return
	}
	SendMail(to, subject, body)
}
