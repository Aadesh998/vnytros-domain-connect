package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mail"
)

func (r *Register) HandleEmail(ctx context.Context, body []byte) error {
	var job jobs.EmailJob
	if err := json.Unmarshal(body, &job); err != nil {
		return Permanent(fmt.Errorf("unmarshal: %w", err))
	}
	if job.To == "" || job.Template == "" {
		return Permanent(fmt.Errorf("invalid email job: %+v", job))
	}

	htmlBody := mail.RenderTemplate(job.Template, job.Vars)
	if htmlBody == "" {
		return Permanent(fmt.Errorf("template %q rendered empty", job.Template))
	}

	mail.SendMail(job.To, job.Subject, htmlBody)
	return nil
}
