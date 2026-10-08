export interface Domain {
  id: number;
  user_id: number;
  domain_name: string;
  status: string;
  ip?: string;
  target?: string;
  text_record?: string;
  session?: string;
  created_at: string;
  updated_at: string;
}

export interface VerifyDomainResponse {
  domain: string;
  status: string;
  message?: string;
}
