/**
 * Wire types. These mirror the API's JSON exactly, including its snake_case
 * field names, so there is no mapping layer that can silently drift from the Go
 * structs in server/internal/views.
 */

/** A DNS provider recognised by nameserver pattern matching. */
export interface Provider {
  id: number;
  name: string;
  ns_patterns: unknown;
  logo: string;
  domain_connect_support: boolean;
  created_at: string;
  updated_at: string;
}

/** `GET /v1/detect` */
export interface DetectResult {
  domain: string;
  nameservers: string[];
  /** `null` when the nameservers match no known provider. */
  provider: Provider | null;
}

export interface DnsRecordSet {
  type: string;
  values: string[];
  error?: string;
}

/** `GET /v1/dns/lookup` */
export interface DnsLookupResult {
  domain: string;
  records: DnsRecordSet[];
}

export type DomainStatusValue = "pending" | "ip_applied" | "completed" | (string & {});

/** A stored domain row. `GET /v1/connect/domains` */
export interface Domain {
  id: number;
  user_id: number;
  domain_name: string;
  status: DomainStatusValue;
  ip: string;
  target: string;
  text_record: string;
  session: string;
  created_at: string;
  updated_at: string;
}

/**
 * DNS providers Vnytros can write records to with credentials you supply:
 * AWS Route 53 and Hostinger. These slugs are the exact strings
 * `dns/factory.GetProviderFromJSON` switches on — `"route53"` is rejected as
 * unsupported. Only record creation is implemented server-side; deleting,
 * listing and testing records through a provider are not yet.
 */
export type DirectProviderSlug = "aws" | "hostinger";

/** AWS Route 53. The IAM user needs `route53:ChangeResourceRecordSets`. */
export interface AwsProviderConfig {
  accessKey: string;
  secretKey: string;
}

export interface HostingerProviderConfig {
  apiToken: string;
}

export type DirectProviderConfig =
  | AwsProviderConfig
  | HostingerProviderConfig;

interface DirectConnectBase {
  domain: string;
  ip?: string;
  target?: string;
  txt?: string;
}

/**
 * A discriminated union, so the credentials must match the provider. The server
 * unmarshals into Go structs that carry no JSON tags, which makes snake_case
 * keys silently unmarshal to empty and fail as "AWS Access Key is empty" — the
 * camelCase keys below are the ones that actually bind.
 */
export type ConnectDirectParams =
  | (DirectConnectBase & { provider: "aws"; providerConfig: AwsProviderConfig })
  | (DirectConnectBase & { provider: "hostinger"; providerConfig: HostingerProviderConfig });

/** `POST /v1/connect/direct` */
export interface ConnectDirectResult {
  message: string;
  domain: string;
}

export interface RecordStatus {
  type: string;
  host: string;
  expected: string;
  current: string[];
  status: boolean;
}

/** `POST /v1/status` */
export interface DomainStatusResult {
  domain: string;
  is_configured: boolean;
  records: RecordStatus[];
}

export interface StatusParams {
  domain: string;
  ip?: string;
  target?: string;
  txt?: string;
}

/** `POST /v1/connect/verify` */
export interface DomainVerifyResult {
  domain: string;
  status: DomainStatusValue;
  ip_matched: boolean;
  txt_matched: boolean;
  cname_matched: boolean;
  message: string;
}

/* ---------------------------------------------------------------- tools --- */

export type FindingStatus = "pass" | "warn" | "fail" | "info";
export type FindingSeverity = "critical" | "high" | "medium" | "low" | "info";

export interface Finding {
  id: string;
  title: string;
  status: FindingStatus;
  severity: FindingSeverity;
  detail: string;
  remediation?: string;
  evidence?: unknown;
}

/** The uniform envelope every `/v1/tools/*` endpoint returns. */
export interface Report {
  tool: string;
  target: string;
  verdict: FindingStatus;
  /** 0-100, reduced per failed finding by severity. */
  score: number;
  /** "A+" through "F", derived from `score`. */
  grade: string;
  summary: string;
  findings: Finding[];
  raw?: unknown;
  timing_ms: number;
  checked_at: string;
}

/* ------------------------------------------------------------- webhooks --- */

export interface DomainConnectedEventData {
  domain: string;
  user_id: number;
}

export interface DomainErrorEventData {
  domain: string;
  error: string;
  message: string;
}

/** The JSON body the API POSTs to your webhook URL. */
export type VnytrosEvent =
  | { event: "domain.connected"; data: DomainConnectedEventData; timestamp: string }
  | { event: "domain.error"; data: DomainErrorEventData; timestamp: string }
  | { event: string & {}; data: Record<string, unknown>; timestamp: string };
