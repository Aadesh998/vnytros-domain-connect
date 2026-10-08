import { Badge } from "@/components/ui/badge"
import type { CampaignStatus, RecipientStatus } from "@/lib/mail/types"

type BadgeVariant = "default" | "secondary" | "destructive" | "outline"

const CAMPAIGN_LABELS: Record<CampaignStatus, string> = {
  draft: "Draft",
  queued: "Queued",
  in_progress: "Sending",
  completed: "Completed",
  failed: "Failed",
}

const CAMPAIGN_VARIANTS: Record<CampaignStatus, BadgeVariant> = {
  draft: "outline",
  queued: "secondary",
  in_progress: "default",
  completed: "default",
  failed: "destructive",
}

export function CampaignStatusBadge({ status }: { status: CampaignStatus }) {
  return (
    <Badge variant={CAMPAIGN_VARIANTS[status] ?? "secondary"}>
      {CAMPAIGN_LABELS[status] ?? status}
    </Badge>
  )
}

const RECIPIENT_VARIANTS: Record<RecipientStatus, BadgeVariant> = {
  queued: "outline",
  sent: "default",
  failed: "destructive",
}

export function RecipientStatusBadge({ status }: { status: RecipientStatus }) {
  return (
    <Badge variant={RECIPIENT_VARIANTS[status] ?? "secondary"}>{status}</Badge>
  )
}
