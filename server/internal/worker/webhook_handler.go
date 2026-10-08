package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/models"

	"gorm.io/gorm"
)

const (
	webhookHTTPTimeout = 15 * time.Second
	webhookMaxAttempts = 5
	webhookBodyLimit   = 4 * 1024
)

var webhookHTTPClient = &http.Client{Timeout: webhookHTTPTimeout}

func (r *Register) HandleWebhook(ctx context.Context, body []byte) error {
	var job jobs.WebhookJob
	if err := json.Unmarshal(body, &job); err != nil {
		return Permanent(fmt.Errorf("unmarshal: %w", err))
	}
	if job.URL == "" || job.EventLogID == 0 {
		return Permanent(fmt.Errorf("invalid job: %+v", job))
	}

	logRow, err := r.webhookLogRepo.GetByID(job.EventLogID)
	if err != nil {
		return Retryable(fmt.Errorf("load event log: %w", err))
	}
	if logRow == nil {
		return Permanent(fmt.Errorf("event log %d not found", job.EventLogID))
	}
	if logRow.Status == models.WebhookDeliveredStatus {
		return nil
	}
	if logRow.Attempts >= webhookMaxAttempts {
		_ = r.webhookLogRepo.Update(job.EventLogID, map[string]any{
			"status": models.WebhookFailedStatus,
		})
		return Permanent(fmt.Errorf("max attempts reached"))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, job.URL,
		bytes.NewReader([]byte(job.Payload)))
	if err != nil {
		return Permanent(fmt.Errorf("build request: %w", err))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Vnytros-Webhook/1.0")
	req.Header.Set("X-Vnytros-Event", job.Event)

	// Sign the exact bytes we send. Each delivery attempt is signed afresh, so a
	// retry carries a new timestamp and stays inside the receiver's replay window.
	if secret := config.AppConfig.WebhookSigningSecret; secret != "" {
		ts := time.Now().UTC().Unix()
		req.Header.Set("X-Vnytros-Timestamp", strconv.FormatInt(ts, 10))
		req.Header.Set("X-Vnytros-Signature", signWebhook(secret, ts, job.Payload))
	}

	resp, err := webhookHTTPClient.Do(req)
	updates := map[string]any{"attempts": gorm.Expr("attempts + 1")}

	if err != nil {
		updates["status"] = models.WebhookPendingStatus
		updates["last_error"] = err.Error()
		_ = r.webhookLogRepo.Update(job.EventLogID, updates)
		return Retryable(err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, webhookBodyLimit))
	updates["response_status"] = resp.StatusCode
	updates["response_body"] = string(respBody)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		updates["status"] = models.WebhookDeliveredStatus
		updates["last_error"] = ""
		_ = r.webhookLogRepo.Update(job.EventLogID, updates)
		return nil

	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		updates["status"] = models.WebhookFailedStatus
		updates["last_error"] = fmt.Sprintf("client error %d", resp.StatusCode)
		_ = r.webhookLogRepo.Update(job.EventLogID, updates)
		return Permanent(fmt.Errorf("client error %d", resp.StatusCode))

	default:
		updates["status"] = models.WebhookPendingStatus
		updates["last_error"] = fmt.Sprintf("server error %d", resp.StatusCode)
		_ = r.webhookLogRepo.Update(job.EventLogID, updates)
		return Retryable(fmt.Errorf("server error %d", resp.StatusCode))
	}
}

// signWebhook returns the value for the X-Vnytros-Signature header. The signed
// string is "<timestamp>.<payload>", so a captured body cannot be replayed under
// a fresh timestamp. The v1= prefix leaves room to rotate the scheme later
// without breaking receivers that pin to a version.
func signWebhook(secret string, timestamp int64, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(mac, "%d.%s", timestamp, payload)
	return "t=" + strconv.FormatInt(timestamp, 10) + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}
