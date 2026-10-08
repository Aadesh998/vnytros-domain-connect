package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/models"
)

const verifyMaxAttempts = 20

func (r *Register) HandleVerifyDomain(ctx context.Context, body []byte) error {
	var job jobs.VerifyDomainJob
	if err := json.Unmarshal(body, &job); err != nil {
		return Permanent(fmt.Errorf("unmarshal: %w", err))
	}
	if job.DomainID == 0 {
		return Permanent(fmt.Errorf("invalid job: %+v", job))
	}
	if job.Attempt >= verifyMaxAttempts {
		return Permanent(fmt.Errorf("max verification attempts reached"))
	}

	domain, err := r.domainRepo.GetByID(job.DomainID)
	if err != nil {
		return Retryable(fmt.Errorf("load domain: %w", err))
	}
	if domain == nil {
		return Permanent(fmt.Errorf("domain %d not found", job.DomainID))
	}
	if domain.Status == models.DomainStatusCompleted {
		return nil
	}

	resolver := &net.Resolver{}
	ips, err := resolver.LookupHost(ctx, domain.DomainName)
	if err != nil {
		return Retryable(fmt.Errorf("dns lookup: %w", err))
	}

	matched := false
	for _, ip := range ips {
		if ip == domain.IP {
			matched = true
			break
		}
	}
	if !matched {
		return Retryable(fmt.Errorf("ip not yet propagated for %s", domain.DomainName))
	}

	if err := r.domainRepo.UpdateStatus(domain.ID, models.DomainStatusCompleted); err != nil {
		return Retryable(fmt.Errorf("update status: %w", err))
	}
	return nil
}
