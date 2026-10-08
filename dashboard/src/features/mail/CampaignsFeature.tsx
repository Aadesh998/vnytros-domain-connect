import { useState } from "react"
import { Link } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  AlertTriangle,
  BarChart3,
  Loader2,
  Mail,
  Plus,
  Send,
  Trash2,
  Upload,
} from "lucide-react"
import { useCampaigns } from "@/hooks/useCampaigns"
import { useMailTemplates } from "@/hooks/useMailTemplates"
import { useSmtpConfigs } from "@/hooks/useSmtpConfigs"
import type { Campaign } from "@/lib/mail/types"
import { CampaignStatusBadge } from "./status"
import { formatPercent } from "./format"

export function MailCampaignsFeature() {
  const [tab, setTab] = useState<"sent" | "drafts">("sent")
  const sent = useCampaigns("sent")
  const drafts = useCampaigns("drafts")
  const { all: templates, isLoading: templatesLoading } = useMailTemplates()
  const { configs, defaultId } = useSmtpConfigs()

  const active = tab === "sent" ? sent : drafts

  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [name, setName] = useState("")
  const [description, setDescription] = useState("")
  const [templateId, setTemplateId] = useState<number | "">("")
  const [createError, setCreateError] = useState("")
  const [isCreating, setIsCreating] = useState(false)

  const [sendTarget, setSendTarget] = useState<Campaign | null>(null)
  const [audience, setAudience] = useState<File | null>(null)
  const [senderId, setSenderId] = useState<number | "">("")
  const [sendError, setSendError] = useState("")
  const [sendNotice, setSendNotice] = useState("")
  const [isSending, setIsSending] = useState(false)

  const resetCreate = () => {
    setName("")
    setDescription("")
    setTemplateId("")
    setCreateError("")
  }

  const handleCreate = async () => {
    if (!name.trim()) {
      setCreateError("Give the campaign a name")
      return
    }
    if (templateId === "") {
      setCreateError("Choose a template")
      return
    }

    setIsCreating(true)
    setCreateError("")
    try {
      await drafts.create({
        campaign_name: name.trim(),
        description: description.trim(),
        template_id: Number(templateId),
        audience_data_source: "csv",
      })
      setIsCreateOpen(false)
      resetCreate()
      setTab("drafts")
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : "Could not create campaign")
    } finally {
      setIsCreating(false)
    }
  }

  const openSend = (campaign: Campaign) => {
    setSendTarget(campaign)
    setAudience(null)
    setSenderId(campaign.smtp_config_id || defaultId || "")
    setSendError("")
    setSendNotice("")
  }

  const handleSend = async () => {
    if (!sendTarget) return
    if (!audience) {
      setSendError("Upload a CSV of recipient addresses")
      return
    }

    setIsSending(true)
    setSendError("")
    try {
      const result = await drafts.send(
        sendTarget.id,
        audience,
        senderId === "" ? undefined : Number(senderId),
      )
      await sent.refresh()
      setSendTarget(null)
      const skipped = result.skipped_invalid
        ? ` ${result.skipped_invalid} invalid address${result.skipped_invalid === 1 ? "" : "es"} skipped.`
        : ""
      setSendNotice(
        `${result.queued_emails} email${result.queued_emails === 1 ? "" : "s"} queued across ${result.batches} batch${result.batches === 1 ? "" : "es"}.${skipped}`,
      )
      setTab("sent")
    } catch (err) {
      setSendError(err instanceof Error ? err.message : "Could not queue campaign")
    } finally {
      setIsSending(false)
    }
  }

  const handleDelete = async (campaign: Campaign) => {
    try {
      await (campaign.status === "draft" ? drafts : sent).remove(campaign.id)
    } catch {
      // The list refresh will show the row still present.
    }
  }

  const noSenders = configs.length === 0

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-vn-text">Campaigns</h1>
          <p className="mt-1 text-sm text-vn-text-3">
            Send templated email to a CSV audience and track how it performs.
          </p>
        </div>
        <Button onClick={() => setIsCreateOpen(true)} disabled={templatesLoading}>
          <Plus className="size-4" />
          New campaign
        </Button>
      </div>

      {noSenders && (
        <div className="flex items-start gap-3 rounded-xl border border-vn-warn/30 bg-vn-warn-soft p-4 text-sm">
          <AlertTriangle className="mt-0.5 size-4 shrink-0 text-vn-warn" />
          <div>
            <p className="font-medium text-vn-text">No sender configured</p>
            <p className="mt-0.5 text-vn-text-3">
              Add SMTP credentials in{" "}
              <Link to="/mail/settings" className="text-vn-accent hover:underline">
                Mail settings
              </Link>{" "}
              before sending a campaign.
            </p>
          </div>
        </div>
      )}

      {sendNotice && (
        <div className="rounded-xl border border-vn-success/30 bg-vn-success-soft p-4 text-sm text-vn-text">
          {sendNotice}
        </div>
      )}

      <Tabs value={tab} onValueChange={(v) => setTab(v as "sent" | "drafts")}>
        <TabsList>
          <TabsTrigger value="sent">Sent</TabsTrigger>
          <TabsTrigger value="drafts">Drafts</TabsTrigger>
        </TabsList>

        <TabsContent value={tab}>
          {active.error && (
            <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
              {active.error}
            </div>
          )}

          {active.isLoading && active.campaigns.length === 0 ? (
            <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
              <Loader2 className="size-4 animate-spin" />
              Loading campaigns…
            </div>
          ) : active.campaigns.length === 0 ? (
            <div className="rounded-xl border border-vn-hairline bg-vn-surface p-8 text-center">
              <Mail className="mx-auto size-6 text-vn-text-4" />
              <p className="mt-3 text-sm font-medium text-vn-text">
                {tab === "drafts" ? "No drafts yet" : "Nothing sent yet"}
              </p>
              <p className="mt-1 text-sm text-vn-text-3">
                {tab === "drafts"
                  ? "Create a campaign to get started."
                  : "Queued and completed campaigns appear here."}
              </p>
            </div>
          ) : (
            <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Campaign</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="text-right">Sent</TableHead>
                    <TableHead className="text-right">Failed</TableHead>
                    <TableHead className="text-right">Open rate</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {active.campaigns.map((campaign) => (
                    <TableRow key={campaign.id}>
                      <TableCell>
                        <p className="font-medium text-vn-text">
                          {campaign.campaign_name}
                        </p>
                        {campaign.description && (
                          <p className="mt-0.5 line-clamp-1 text-xs text-vn-text-3">
                            {campaign.description}
                          </p>
                        )}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <CampaignStatusBadge status={campaign.status} />
                          {campaign.status === "in_progress" && (
                            <span className="text-xs text-vn-text-3">
                              {formatPercent(campaign.progress_percentage)}
                              {campaign.estimated_time &&
                                ` · ${campaign.estimated_time} left`}
                            </span>
                          )}
                        </div>
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-vn-text-2">
                        {campaign.sent_emails}
                        <span className="text-vn-text-4">
                          /{campaign.total_emails}
                        </span>
                      </TableCell>
                      <TableCell className="text-right tabular-nums">
                        {campaign.failed_emails > 0 ? (
                          <span className="text-vn-danger">
                            {campaign.failed_emails}
                          </span>
                        ) : (
                          <span className="text-vn-text-4">0</span>
                        )}
                      </TableCell>
                      <TableCell className="text-right tabular-nums text-vn-text-2">
                        {formatPercent(campaign.open_rate)}
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center justify-end gap-1.5">
                          {campaign.status === "draft" && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => openSend(campaign)}
                              disabled={noSenders}
                            >
                              <Send className="size-3.5" />
                              Send
                            </Button>
                          )}
                          {campaign.status !== "draft" && (
                            <Button variant="outline" size="sm" asChild>
                              <Link to={`/mail/campaigns/${campaign.id}`}>
                                <BarChart3 className="size-3.5" />
                                Report
                              </Link>
                            </Button>
                          )}
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label="Delete campaign"
                            onClick={() => void handleDelete(campaign)}
                          >
                            <Trash2 className="size-3.5 text-vn-danger" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}

          {active.hasMore && (
            <div className="mt-3 flex justify-center">
              <Button
                variant="outline"
                size="sm"
                onClick={() => void active.loadMore()}
                disabled={active.isLoading}
              >
                {active.isLoading && <Loader2 className="size-4 animate-spin" />}
                Load more
              </Button>
            </div>
          )}
        </TabsContent>
      </Tabs>

      <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>New campaign</DialogTitle>
            <DialogDescription>
              Campaigns start as drafts. You upload the audience when you send.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">Name</label>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="October product update"
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Description
              </label>
              <Textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Optional internal note"
                rows={2}
              />
            </div>
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Template
              </label>
              <select
                value={templateId}
                onChange={(e) =>
                  setTemplateId(e.target.value === "" ? "" : Number(e.target.value))
                }
                className="w-full rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
              >
                <option value="">Select a template…</option>
                {templates.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name} {t.status === "draft" ? "(draft)" : ""}
                  </option>
                ))}
              </select>
              {templates.length === 0 && !templatesLoading && (
                <p className="mt-1.5 text-xs text-vn-text-3">
                  No templates yet —{" "}
                  <Link to="/mail/templates" className="text-vn-accent hover:underline">
                    create one first
                  </Link>
                  .
                </p>
              )}
            </div>
            {createError && (
              <p className="text-sm text-vn-danger">{createError}</p>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
              Cancel
            </Button>
            <Button onClick={() => void handleCreate()} disabled={isCreating}>
              {isCreating && <Loader2 className="size-4 animate-spin" />}
              Create draft
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={sendTarget !== null}
        onOpenChange={(open) => !open && setSendTarget(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Send “{sendTarget?.campaign_name}”</DialogTitle>
            <DialogDescription>
              Upload a CSV whose first column holds email addresses. A header row
              named “email” is optional.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Audience CSV
              </label>
              <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-dashed border-vn-hairline-2 bg-vn-surface-2 px-3 py-3 text-sm text-vn-text-2 hover:border-vn-accent/40">
                <Upload className="size-4 text-vn-text-3" />
                <span className="truncate">
                  {audience ? audience.name : "Choose a .csv file"}
                </span>
                <input
                  type="file"
                  accept=".csv,text/csv"
                  className="sr-only"
                  onChange={(e) => setAudience(e.target.files?.[0] ?? null)}
                />
              </label>
            </div>

            <div>
              <label className="mb-1.5 block text-sm text-vn-text-2">
                Send from
              </label>
              <select
                value={senderId}
                onChange={(e) =>
                  setSenderId(e.target.value === "" ? "" : Number(e.target.value))
                }
                className="w-full rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
              >
                <option value="">Use my default sender</option>
                {configs
                  .filter((c) => c.active)
                  .map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.label} — {c.from_email}
                    </option>
                  ))}
              </select>
            </div>

            <div className="rounded-lg border border-vn-hairline bg-vn-surface-2 p-3 text-xs text-vn-text-3">
              Sending is queued and runs in the background, so this returns
              immediately. Track progress on the campaign report.
            </div>

            {sendError && <p className="text-sm text-vn-danger">{sendError}</p>}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setSendTarget(null)}>
              Cancel
            </Button>
            <Button onClick={() => void handleSend()} disabled={isSending}>
              {isSending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Send className="size-4" />
              )}
              Queue campaign
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
