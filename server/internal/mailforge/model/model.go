// Package model names the mail_* tables for the mailforge half of this repo.
//
// These used to be hand-copied struct definitions that had to stay
// column-for-column identical to internal/models/mail.go — Go could not share a
// type across the module boundary while mailforge lived in its own repo. That
// boundary is gone, so they are now aliases of the api's canonical structs:
// there is exactly one definition of each mail_* table, cmd/migration still
// owns the schema, and the two halves can no longer drift apart.
package model

import "domain-connect-backend/internal/models"

type (
	Template   = models.MailTemplate
	SmtpConfig = models.MailSmtpConfig
	Campaign   = models.MailCampaign
	Recipient  = models.MailRecipient
	Track      = models.MailTrack
	Branding   = models.MailBranding
)

const (
	CampaignStatusDraft      = models.CampaignStatusDraft
	CampaignStatusQueued     = models.CampaignStatusQueued
	CampaignStatusInProgress = models.CampaignStatusInProgress
	CampaignStatusCompleted  = models.CampaignStatusCompleted
	CampaignStatusFailed     = models.CampaignStatusFailed
)

const (
	RecipientStatusQueued = models.RecipientStatusQueued
	RecipientStatusSent   = models.RecipientStatusSent
	RecipientStatusFailed = models.RecipientStatusFailed
)

const TrackEventOpen = models.TrackEventOpen

// Encryption modes for SmtpConfig.Encryption. Only mailforge validates these,
// so they have no counterpart in internal/models.
const (
	EncryptionStartTLS = "starttls"
	EncryptionSSL      = "ssl"
	EncryptionNone     = "none"
)
