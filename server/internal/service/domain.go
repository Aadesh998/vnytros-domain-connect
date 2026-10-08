package service

import (
	"context"
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/dns"
	"domain-connect-backend/internal/dns/factory"
	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/mail"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/queue"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/utils"
	views "domain-connect-backend/internal/views"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DomainService interface {
	DetectProvider(domainName string) (string, []string, *models.Providers, error)
	LookupDNS(domainName string) (*views.DNSLookupResponse, error)
	GetDomainConnectSettings(domainName, dnsProvider string) (*views.AutoConnectResponse, error)
	GetPublicKey() ([]byte, error)
	GetTemplate() (map[string]interface{}, error)
	GetDiscovery() (map[string]interface{}, error)
	CheckDomainStatus(req views.DomainStatusRequest) (*views.DomainStatusResponse, error)
	VerifyDomain(userID uint, domainName string) (*views.DomainVerifyResponse, error)
	ListUserDomains(userID uint) ([]*models.Domains, error)
	GetUserDomain(userID uint, domainName string) (*models.Domains, error)
	DirectProviderConnect(ctx context.Context, userID uint, req views.DirectConnectRequest) error
}

type domainService struct {
	repo           repository.ProviderRepository
	domainRepo     repository.DomainRepository
	userRepo       repository.UserRepository
	apiKeyRepo     repository.ApiKeyRepository
	webhookLogRepo repository.WebhookEventLogRepository
	providerMap    map[string]*models.Providers
	nextUpdate     time.Time
	cacheMu        sync.RWMutex
}

const providerCacheTTL = 12 * time.Hour

func NewDomainService(
	repo repository.ProviderRepository,
	domainRepo repository.DomainRepository,
	userRepo repository.UserRepository,
	apiKeyRepo repository.ApiKeyRepository,
	webhookLogRepo repository.WebhookEventLogRepository,
) DomainService {
	return &domainService{
		repo:           repo,
		domainRepo:     domainRepo,
		userRepo:       userRepo,
		apiKeyRepo:     apiKeyRepo,
		webhookLogRepo: webhookLogRepo,
	}
}

func (s *domainService) emitWebhook(userID uint, event string, data map[string]any) {
	if s.webhookLogRepo == nil {
		return
	}
	apiKey, err := s.apiKeyRepo.GetByUserID(userID)
	if err != nil || apiKey == nil || apiKey.WebHook == "" {
		return
	}

	payload, err := json.Marshal(map[string]any{
		"event":     event,
		"data":      data,
		"timestamp": time.Now().UTC(),
	})
	if err != nil {
		log.Printf("emitWebhook: marshal payload: %v", err)
		return
	}

	logRow := &models.WebHookEventLogs{
		WebHook:  apiKey.WebHook,
		Event:    event,
		Status:   models.WebhookPendingStatus,
		Payload:  string(payload),
		ApiKeyID: apiKey.ID,
		UserID:   userID,
	}
	if err := s.webhookLogRepo.Create(logRow); err != nil {
		log.Printf("emitWebhook: persist log: %v", err)
		return
	}

	if err := queue.EnqueueWebhook(context.Background(), logRow.ID, apiKey.WebHook, event, string(payload)); err != nil {
		log.Printf("emitWebhook: enqueue: %v", err)
	}
}

func (s *domainService) GetPublicKey() ([]byte, error) {
	data, err := os.ReadFile(utils.PublicKeyFile)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *domainService) GetTemplate() (map[string]interface{}, error) {
	dc := config.AppConfig.DCProviderDomain
	if dc == "" {
		return nil, fmt.Errorf("domain connect template: DC_PROVIDER_DOMAIN is not set")
	}
	templatebytedata, err := os.ReadFile(utils.TemplateFile(dc))
	if err != nil {
		return nil, err
	}

	var response map[string]interface{}
	err = json.Unmarshal(templatebytedata, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (s *domainService) GetDiscovery() (map[string]interface{}, error) {
	cfg := config.AppConfig
	discovery := map[string]interface{}{
		"providerId":   cfg.DCProviderDomain,
		"providerName": cfg.ProductName,
		"serviceId":    "custom-domain",
		"urlAPI":       cfg.BaseURL,
		"width":        750,
		"height":       750,
		"object":       false,
	}
	return discovery, nil
}

func (s *domainService) getProviderMap() (map[string]*models.Providers, error) {
	s.cacheMu.RLock()
	if time.Now().Before(s.nextUpdate) && s.providerMap != nil {
		m := s.providerMap
		s.cacheMu.RUnlock()
		return m, nil
	}
	s.cacheMu.RUnlock()

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	if time.Now().Before(s.nextUpdate) && s.providerMap != nil {
		return s.providerMap, nil
	}

	providers, err := s.repo.GetAllProviders()
	if err != nil {
		return nil, err
	}

	newMap := make(map[string]*models.Providers)
	for _, p := range providers {
		var patterns []string
		if err := json.Unmarshal(p.NSPatterns, &patterns); err != nil {
			log.Printf("Error unmarshaling patterns for provider %s: %v", p.Name, err)
			continue
		}

		for _, pattern := range patterns {
			newMap[strings.ToLower(pattern)] = p
		}
	}

	s.providerMap = newMap
	s.nextUpdate = time.Now().Add(providerCacheTTL)
	return s.providerMap, nil
}

func (s *domainService) DetectProvider(domainName string) (string, []string, *models.Providers, error) {
	nss, err := net.LookupNS(domainName)
	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return "", nil, nil, errorz.ErrDomainNotFound
		}
		return "", nil, nil, err
	}

	if len(nss) == 0 {
		return "", nil, nil, errorz.ErrDomainNotFound
	}

	var nameservers []string
	for _, ns := range nss {
		nameservers = append(nameservers, strings.TrimSuffix(ns.Host, "."))
	}

	pMap, err := s.getProviderMap()
	if err != nil {
		return "", nil, nil, err
	}

	var detectedProvider *models.Providers
OuterLoop:
	for _, ns := range nameservers {
		ns = strings.ToLower(ns)
		parts := strings.Split(ns, ".")

		for i := 0; i < len(parts); i++ {
			suffix := strings.Join(parts[i:], ".")
			if p, ok := pMap[suffix]; ok {
				detectedProvider = p
				break OuterLoop
			}
		}
	}

	return domainName, nameservers, detectedProvider, nil
}

func (s *domainService) LookupDNS(domainName string) (*views.DNSLookupResponse, error) {
	domainName = strings.ToLower(strings.TrimSpace(domainName))
	domainName = strings.TrimSuffix(domainName, ".")
	if domainName == "" {
		return nil, errorz.ErrInvalidInput
	}

	if _, err := net.LookupNS(domainName); err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return nil, errorz.ErrDomainNotFound
		}
	}

	type result struct {
		index int
		set   views.DNSRecordSet
	}

	lookups := []func() views.DNSRecordSet{
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "A"}
			ips, err := net.LookupIP(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			for _, ip := range ips {
				if ip.To4() != nil {
					set.Values = append(set.Values, ip.String())
				}
			}
			return set
		},
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "AAAA"}
			ips, err := net.LookupIP(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			for _, ip := range ips {
				if ip.To4() == nil {
					set.Values = append(set.Values, ip.String())
				}
			}
			return set
		},
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "MX"}
			mxs, err := net.LookupMX(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			for _, mx := range mxs {
				set.Values = append(set.Values, fmt.Sprintf("%d %s", mx.Pref, strings.TrimSuffix(mx.Host, ".")))
			}
			return set
		},
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "TXT"}
			txts, err := net.LookupTXT(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			set.Values = txts
			return set
		},
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "NS"}
			nss, err := net.LookupNS(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			for _, ns := range nss {
				set.Values = append(set.Values, strings.TrimSuffix(ns.Host, "."))
			}
			return set
		},
		func() views.DNSRecordSet {
			set := views.DNSRecordSet{Type: "CNAME"}
			cname, err := net.LookupCNAME(domainName)
			if err != nil {
				set.Error = dnsErrorMessage(err)
				return set
			}
			cname = strings.TrimSuffix(cname, ".")
			if cname != "" && cname != domainName {
				set.Values = append(set.Values, cname)
			}
			return set
		},
	}

	results := make([]views.DNSRecordSet, len(lookups))
	ch := make(chan result, len(lookups))
	for i, fn := range lookups {
		go func(i int, fn func() views.DNSRecordSet) {
			ch <- result{index: i, set: fn()}
		}(i, fn)
	}
	for range lookups {
		r := <-ch
		results[r.index] = r.set
	}

	return &views.DNSLookupResponse{
		Domain:  domainName,
		Records: results,
	}, nil
}

func dnsErrorMessage(err error) string {
	if dnsErr, ok := err.(*net.DNSError); ok {
		if dnsErr.IsNotFound {
			return "no records found"
		}
		if dnsErr.IsTimeout {
			return "lookup timed out"
		}
	}
	return "lookup failed"
}

func (s *domainService) GetDomainConnectSettings(domainName, dnsProvider string) (*views.AutoConnectResponse, error) {
	_domainConnectURL := fmt.Sprintf("_domainconnect.%s", domainName)
	txt_records, err := net.LookupTXT(_domainConnectURL)

	if err != nil {
		if dnsErr, ok := err.(*net.DNSError); ok && dnsErr.IsNotFound {
			return nil, errorz.ErrDomainNotFound
		}
		return nil, err
	}

	if len(txt_records) == 0 {
		return nil, errorz.ErrDomainNotFound
	}

	apiHost := strings.Trim(txt_records[0], "\"")
	settingsURL := fmt.Sprintf("https://%s/v2/%s/settings", apiHost, domainName)

	resp, err := http.Get(settingsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("settings API returned status %d", resp.StatusCode)
	}

	var settings views.DomainConnectSettings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, err
	}

	response := &views.AutoConnectResponse{
		ProviderId:   settings.ProviderId,
		ProviderName: settings.ProviderName,
		SettingsURL:  settingsURL,
	}

	if dnsProvider == "Cloudflare" {
		response.URL = settings.URLSyncUX
		response.Type = "sync"
	} else {
		if settings.URLAsyncUX != "" {
			response.URL = settings.URLAsyncUX
			response.Type = "async"
		} else {
			response.URL = settings.URLSyncUX
			response.Type = "sync"
		}
	}

	return response, nil
}

func (s *domainService) GetUserDomain(userID uint, domainName string) (*models.Domains, error) {
	d, err := s.domainRepo.GetByDomain(strings.ToLower(strings.TrimSpace(domainName)))
	if err != nil {
		return nil, err
	}
	if d == nil || d.UserID != userID {
		return nil, errorz.ErrNotFound
	}
	return d, nil
}

func (s *domainService) ListUserDomains(userID uint) ([]*models.Domains, error) {
	return s.domainRepo.GetByUserID(userID)
}

func (s *domainService) VerifyDomain(userID uint, domainName string) (*views.DomainVerifyResponse, error) {
	domainName = strings.ToLower(strings.TrimSpace(domainName))
	if domainName == "" {
		return nil, errorz.ErrInvalidInput
	}

	record, err := s.domainRepo.GetByDomain(domainName)
	if err != nil {
		return nil, err
	}
	if record == nil || record.UserID != userID {
		return nil, errorz.ErrNotFound
	}

	ipMatched := record.IP == ""
	if record.IP != "" {
		ips, err := net.LookupIP(domainName)
		if err == nil {
			for _, ip := range ips {
				if ip.String() == record.IP {
					ipMatched = true
					break
				}
			}
		}
	}

	cnameMatched := record.Target == ""
	if record.Target != "" {
		cname, err := net.LookupCNAME("www." + domainName)
		if err == nil {
			cname = strings.TrimSuffix(cname, ".")
			if strings.EqualFold(cname, strings.TrimSuffix(record.Target, ".")) {
				cnameMatched = true
			}
		}
	}

	txtMatched := record.TextRecord == ""
	if record.TextRecord != "" {
		expected := record.TextRecord
		if !strings.HasPrefix(expected, "vnytros-verify=") {
			expected = "vnytros-verify=" + expected
		}
		txts, err := net.LookupTXT(domainName)
		if err == nil {
			for _, txt := range txts {
				if txt == expected || txt == record.TextRecord {
					txtMatched = true
					break
				}
			}
		}
	}

	newStatus := models.DomainStatusPending
	switch {
	case txtMatched && ipMatched && cnameMatched:
		newStatus = models.DomainStatusCompleted
	case ipMatched:
		newStatus = models.DomainStatusIPApplied
	}

	previousStatus := record.Status
	if newStatus != previousStatus {
		record.Status = newStatus
		if err := s.domainRepo.Update(record); err != nil {
			log.Printf("ERROR: Failed to update domain status for %s: %v", domainName, err)
			return nil, err
		}
	}

	if newStatus == models.DomainStatusCompleted && previousStatus != models.DomainStatusCompleted {
		if err := s.apiKeyRepo.IncrementDomainCount(record.UserID); err != nil {
			log.Printf("Failed to update API key domain count: %v", err)
		}

		user, err := s.userRepo.GetByID(record.UserID)
		if err == nil || user != nil {
			mail.Enqueue(user.Email,
				fmt.Sprintf("Your domain %s is now connected", record.DomainName),
				"domain.html",
				map[string]string{
					"name":   user.Name,
					"domain": record.DomainName,
				})

			s.emitWebhook(record.UserID, "domain.connected", map[string]any{
				"domain":  record.DomainName,
				"user_id": record.UserID,
			})
		}
	}

	return &views.DomainVerifyResponse{
		Domain:     record.DomainName,
		Status:     record.Status,
		IPMatched:  ipMatched,
		TXTMatched: txtMatched,
		CNAMEMatch: cnameMatched,
		Message:    verifyMessage(record.Status),
	}, nil
}

func verifyMessage(status string) string {
	switch status {
	case models.DomainStatusCompleted:
		return "Domain successfully verified and connected."
	case models.DomainStatusIPApplied:
		return "IP record applied. Waiting for TXT and CNAME to propagate."
	default:
		return "Domain verification still pending. Please ensure DNS records are configured."
	}
}

func (s *domainService) CheckDomainStatus(req views.DomainStatusRequest) (*views.DomainStatusResponse, error) {
	var records []views.RecordStatus
	isConfigured := true

	// Check A Record
	if req.IP != "" {
		ips, err := net.LookupIP(req.Domain)
		var currentIPs []string
		status := false
		if err == nil {
			for _, ip := range ips {
				ipStr := ip.String()
				currentIPs = append(currentIPs, ipStr)
				if ipStr == req.IP {
					status = true
				}
			}
		}
		if !status {
			isConfigured = false
		}
		records = append(records, views.RecordStatus{
			Type:     "A",
			Host:     "@",
			Expected: req.IP,
			Current:  currentIPs,
			Status:   status,
		})
	}

	// Check CNAME Record for www
	if req.Target != "" {
		wwwDomain := "www." + req.Domain
		cname, err := net.LookupCNAME(wwwDomain)
		var currentCNAME []string
		status := false
		if err == nil {
			cname = strings.TrimSuffix(cname, ".")
			currentCNAME = append(currentCNAME, cname)
			if strings.EqualFold(cname, strings.TrimSuffix(req.Target, ".")) {
				status = true
			}
		}
		if !status {
			isConfigured = false
		}
		records = append(records, views.RecordStatus{
			Type:     "CNAME",
			Host:     "www",
			Expected: req.Target,
			Current:  currentCNAME,
			Status:   status,
		})
	}

	// Check TXT Record for verification
	if req.TXT != "" {
		txts, err := net.LookupTXT(req.Domain)
		var currentTXTs []string
		status := false

		expectedTXT := req.TXT
		if !strings.HasPrefix(expectedTXT, "vnytros-verify=") {
			expectedTXT = "vnytros-verify=" + expectedTXT
		}

		if err == nil {
			for _, txt := range txts {
				currentTXTs = append(currentTXTs, txt)
				if txt == expectedTXT {
					status = true
				}
			}
		}
		if !status {
			isConfigured = false
		}
		records = append(records, views.RecordStatus{
			Type:     "TXT",
			Host:     "@",
			Expected: expectedTXT,
			Current:  currentTXTs,
			Status:   status,
		})
	}

	return &views.DomainStatusResponse{
		Domain:       req.Domain,
		IsConfigured: isConfigured,
		Records:      records,
	}, nil
}

func (s *domainService) DirectProviderConnect(ctx context.Context, userID uint, req views.DirectConnectRequest) error {
	provider, parseErr := factory.GetProviderFromJSON(req.Provider, req.ProviderConfig)
	if parseErr != nil {
		return fmt.Errorf("failed to initialize provider: %w", parseErr)
	}

	var records []dns.DNSRecord
	if req.IP != "" {
		records = append(records, dns.DNSRecord{
			Type:  dns.TypeA,
			Name:  "@",
			Value: req.IP,
			TTL:   3600,
		})
	}
	if req.Target != "" {
		records = append(records, dns.DNSRecord{
			Type:  dns.TypeCNAME,
			Name:  "www",
			Value: req.Target,
			TTL:   3600,
		})
	}

	id := uuid.New()
	uuidStr := id.String()
	session := uuidStr

	txtToken := req.TXT
	if txtToken == "" {
		txtToken = "vnytros-" + uuidStr
	}

	records = append(records, dns.DNSRecord{
		Type:  dns.TypeTXT,
		Name:  "@",
		Value: txtToken,
		TTL:   3600,
	})

	if err := provider.AddRecord(ctx, req.Domain, records); err != nil {
		return fmt.Errorf("failed to apply DNS records via %s: %w", req.Provider, err)
	}

	if s.domainRepo != nil {
		record := &models.Domains{
			UserID:     userID,
			DomainName: strings.ToLower(strings.TrimSpace(req.Domain)),
			Status:     models.DomainStatusPending,
			IP:         req.IP,
			Target:     req.Target,
			TextRecord: txtToken,
			Session:    session,
		}
		if err := s.domainRepo.UpsertByDomain(record); err != nil {
			log.Printf("ERROR: Failed to persist domain record for %s: %v", req.Domain, err)
			return err
		}

		if err := queue.EnqueueVerifyDomain(context.Background(), record.ID, 0); err != nil {
			log.Printf("WARN: enqueue verify_domain for %s: %v", record.DomainName, err)
		}
	}

	return nil
}
