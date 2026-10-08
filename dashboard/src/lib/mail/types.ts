export type CampaignStatus =
  | "draft"
  | "queued"
  | "in_progress"
  | "completed"
  | "failed";

export type TemplateStatus = "draft" | "published";

export type RecipientStatus = "queued" | "sent" | "failed";

export type SmtpEncryption = "starttls" | "ssl" | "none";

export interface Campaign {
  id: number;
  campaign_name: string;
  description: string;
  status: CampaignStatus;
  total_emails: number;
  sent_emails: number;
  failed_emails: number;
  opened_emails: number;
  estimated_time: string;
  progress_percentage: number;
  failure_percentage: number;
  open_rate: number;
  template_id: number;
  smtp_config_id: number;
  audience_data_source: string;
  started_at?: string;
  completed_at?: string;
  created_at: string;
}

export interface CampaignPage {
  campaigns: Campaign[] | null;
  next_id?: number;
}

export interface CampaignPayload {
  campaign_name: string;
  description?: string;
  status?: CampaignStatus;
  template_id: number;
  audience_data_source: string;
}

export interface SendCampaignResult {
  campaign: Campaign;
  queued_emails: number;
  batches: number;
  skipped_invalid: number;
  message: string;
}

export interface MailTemplate {
  id: number;
  name: string;
  subject: string;
  status: TemplateStatus;
  body: string;
  created_at: string;
}

export interface TemplatePage {
  templates: MailTemplate[] | null;
  next_id?: number;
}

export interface TemplatePayload {
  name: string;
  subject: string;
  body: string;
  status: TemplateStatus;
}

export interface SmtpConfig {
  id: number;
  label: string;
  from_email: string;
  from_name?: string;
  username: string;
  host: string;
  port: number;
  encryption: SmtpEncryption;
  is_default: boolean;
  active: boolean;
  has_password: boolean;
  last_verified_at?: string;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface SmtpConfigList {
  configs: SmtpConfig[] | null;
  default_id?: number;
}

export interface SmtpConfigPayload {
  label: string;
  from_email: string;
  from_name?: string;
  username?: string;
  /** Omit or leave empty on update to keep the stored password. */
  password?: string;
  host: string;
  port: number;
  encryption?: SmtpEncryption;
  is_default?: boolean;
  active?: boolean;
}

export interface Branding {
  show_watermark: boolean;
  watermark_image_url: string;
  watermark_link_url: string;
  watermark_label: string;
  allowed_image_urls: string[] | null;
}

export interface BrandingPayload {
  show_watermark?: boolean;
  watermark_image_url?: string;
  watermark_link_url?: string;
  watermark_label?: string;
}

export interface TimeSeriesPoint {
  bucket: string;
  opens: number;
  unique_opens: number;
}

export interface AccountAnalytics {
  total_campaigns: number;
  active_campaigns: number;
  total_emails: number;
  sent_emails: number;
  failed_emails: number;
  unique_opens: number;
  delivery_rate: number;
  failure_rate: number;
  open_rate: number;
  timeline: TimeSeriesPoint[] | null;
}

export interface TopRecipient {
  email: string;
  open_count: number;
  last_opened_at?: string;
}

export interface FailureReason {
  reason: string;
  count: number;
}

export interface CampaignAnalytics {
  campaign_id: number;
  campaign_name: string;
  status: CampaignStatus;
  total_emails: number;
  queued_emails: number;
  sent_emails: number;
  failed_emails: number;
  unique_opens: number;
  total_opens: number;
  delivery_rate: number;
  failure_rate: number;
  open_rate: number;
  estimated_time: string;
  started_at?: string;
  completed_at?: string;
  timeline: TimeSeriesPoint[] | null;
  top_recipients: TopRecipient[] | null;
  failure_reasons: FailureReason[] | null;
}

export interface RecipientEntry {
  email: string;
  status: RecipientStatus;
  error_message?: string;
  sent_at?: string;
  first_opened_at?: string;
  last_opened_at?: string;
  open_count: number;
}

export interface RecipientList {
  recipients: RecipientEntry[] | null;
  total: number;
  offset: number;
  limit: number;
}
