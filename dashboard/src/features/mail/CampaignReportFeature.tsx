import { useEffect, useMemo, useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts"
import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  ArrowLeft,
  CheckCircle2,
  Eye,
  Loader2,
  MailX,
  RefreshCw,
  Users,
} from "lucide-react"
import { useCampaignAnalytics } from "@/hooks/useMailAnalytics"
import { campaignsClient } from "@/lib/mail/client"
import type { RecipientEntry, RecipientStatus } from "@/lib/mail/types"
import { StatTile } from "./StatTile"
import { CampaignStatusBadge, RecipientStatusBadge } from "./status"
import { formatDateTime, formatPercent } from "./format"

export function CampaignReportFeature() {
  const { id } = useParams<{ id: string }>()
  const campaignId = id ? Number(id) : null
  const [days, setDays] = useState(30)
  const { analytics, isLoading, error, refresh } = useCampaignAnalytics(
    campaignId,
    days,
  )

  const chartData = useMemo(
    () =>
      (analytics?.timeline ?? []).map((point) => ({
        label: new Date(point.bucket).toLocaleDateString(undefined, {
          month: "short",
          day: "numeric",
        }),
        opens: point.opens,
        unique: point.unique_opens,
      })),
    [analytics?.timeline],
  )

  if (campaignId === null || Number.isNaN(campaignId)) {
    return <p className="text-sm text-vn-danger">Invalid campaign id.</p>
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <Button variant="ghost" size="sm" asChild className="mb-1 -ml-2">
            <Link to="/mail/campaigns">
              <ArrowLeft className="size-3.5" />
              Campaigns
            </Link>
          </Button>
          <div className="flex items-center gap-2.5">
            <h1 className="text-xl font-semibold text-vn-text">
              {analytics?.campaign_name ?? "Campaign report"}
            </h1>
            {analytics && <CampaignStatusBadge status={analytics.status} />}
          </div>
          {analytics?.status === "in_progress" && (
            <p className="mt-1 text-sm text-vn-text-3">
              Sending — {analytics.estimated_time || "calculating"} remaining.
              This page updates itself.
            </p>
          )}
        </div>

        <div className="flex items-center gap-2">
          <select
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            className="rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
          >
            <option value={1}>Last 24 hours</option>
            <option value={7}>Last 7 days</option>
            <option value={30}>Last 30 days</option>
            <option value={90}>Last 90 days</option>
          </select>
          <Button variant="outline" size="sm" onClick={() => void refresh()}>
            <RefreshCw className="size-3.5" />
            Refresh
          </Button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border border-vn-danger/30 bg-vn-danger-soft p-4 text-sm text-vn-danger">
          {error}
        </div>
      )}

      {isLoading && !analytics ? (
        <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
          <Loader2 className="size-4 animate-spin" />
          Loading report…
        </div>
      ) : analytics ? (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <StatTile
              label="Audience"
              value={analytics.total_emails}
              hint={
                analytics.queued_emails > 0
                  ? `${analytics.queued_emails} still queued`
                  : "All processed"
              }
              icon={Users}
            />
            <StatTile
              label="Delivered"
              value={analytics.sent_emails}
              hint={`${formatPercent(analytics.delivery_rate)} of audience`}
              icon={CheckCircle2}
              tone="positive"
            />
            <StatTile
              label="Failed"
              value={analytics.failed_emails}
              hint={`${formatPercent(analytics.failure_rate)} of audience`}
              icon={MailX}
              tone={analytics.failed_emails > 0 ? "negative" : "neutral"}
            />
            <StatTile
              label="Unique opens"
              value={analytics.unique_opens}
              hint={`${formatPercent(analytics.open_rate)} of delivered · ${analytics.total_opens} total`}
              icon={Eye}
            />
          </div>

          <div className="rounded-xl border border-vn-hairline bg-vn-surface p-4">
            <p className="font-mono text-[10px] uppercase tracking-widest text-vn-text-4">
              Opens over time
            </p>
            {chartData.length === 0 ? (
              <p className="py-10 text-center text-sm text-vn-text-3">
                No opens recorded in this window yet.
              </p>
            ) : (
              <div className="mt-3 h-56 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={chartData}>
                    <defs>
                      <linearGradient id="openFill" x1="0" y1="0" x2="0" y2="1">
                        <stop
                          offset="0%"
                          stopColor="currentColor"
                          stopOpacity={0.28}
                          className="text-vn-accent"
                        />
                        <stop
                          offset="100%"
                          stopColor="currentColor"
                          stopOpacity={0}
                          className="text-vn-accent"
                        />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      strokeDasharray="3 3"
                      stroke="currentColor"
                      className="text-vn-hairline"
                      vertical={false}
                    />
                    <XAxis
                      dataKey="label"
                      tick={{ fontSize: 11 }}
                      stroke="currentColor"
                      className="text-vn-text-4"
                      tickLine={false}
                      axisLine={false}
                    />
                    <YAxis
                      allowDecimals={false}
                      tick={{ fontSize: 11 }}
                      stroke="currentColor"
                      className="text-vn-text-4"
                      tickLine={false}
                      axisLine={false}
                      width={32}
                    />
                    <Tooltip
                      contentStyle={{
                        borderRadius: 10,
                        border: "1px solid rgba(255,255,255,0.12)",
                        background: "rgba(20,20,24,0.95)",
                        fontSize: 12,
                      }}
                    />
                    <Area
                      type="monotone"
                      dataKey="opens"
                      name="Opens"
                      stroke="currentColor"
                      className="text-vn-accent"
                      strokeWidth={2}
                      fill="url(#openFill)"
                    />
                    <Area
                      type="monotone"
                      dataKey="unique"
                      name="Unique"
                      stroke="currentColor"
                      className="text-vn-cyan"
                      strokeWidth={1.5}
                      fill="none"
                    />
                  </AreaChart>
                </ResponsiveContainer>
              </div>
            )}
          </div>

          <Tabs defaultValue="recipients">
            <TabsList>
              <TabsTrigger value="recipients">Recipients</TabsTrigger>
              <TabsTrigger value="engaged">Most engaged</TabsTrigger>
              <TabsTrigger value="failures">Failures</TabsTrigger>
            </TabsList>

            <TabsContent value="recipients">
              <RecipientLog campaignId={campaignId} />
            </TabsContent>

            <TabsContent value="engaged">
              {(analytics.top_recipients ?? []).length === 0 ? (
                <EmptyPanel message="No opens recorded yet." />
              ) : (
                <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Recipient</TableHead>
                        <TableHead className="text-right">Opens</TableHead>
                        <TableHead>Last opened</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(analytics.top_recipients ?? []).map((r) => (
                        <TableRow key={r.email}>
                          <TableCell className="text-vn-text">{r.email}</TableCell>
                          <TableCell className="text-right tabular-nums text-vn-text-2">
                            {r.open_count}
                          </TableCell>
                          <TableCell className="text-vn-text-3">
                            {formatDateTime(r.last_opened_at)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </TabsContent>

            <TabsContent value="failures">
              {(analytics.failure_reasons ?? []).length === 0 ? (
                <EmptyPanel message="No failures — every address was accepted." />
              ) : (
                <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Reason</TableHead>
                        <TableHead className="text-right">Count</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {(analytics.failure_reasons ?? []).map((r) => (
                        <TableRow key={r.reason}>
                          <TableCell className="max-w-xl break-words text-vn-text-2">
                            {r.reason}
                          </TableCell>
                          <TableCell className="text-right tabular-nums text-vn-danger">
                            {r.count}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </>
      ) : null}
    </div>
  )
}

function EmptyPanel({ message }: { message: string }) {
  return (
    <div className="rounded-xl border border-vn-hairline bg-vn-surface p-8 text-center text-sm text-vn-text-3">
      {message}
    </div>
  )
}

const PAGE_SIZE = 25

/** Per-address delivery log, paged and filterable by status. */
function RecipientLog({ campaignId }: { campaignId: number }) {
  const [status, setStatus] = useState<RecipientStatus | "">("")
  const [offset, setOffset] = useState(0)
  const [rows, setRows] = useState<RecipientEntry[]>([])
  const [total, setTotal] = useState(0)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      setIsLoading(true)
      setError(null)
      try {
        const data = await campaignsClient.recipients(campaignId, {
          status: status === "" ? undefined : status,
          offset,
          limit: PAGE_SIZE,
        })
        if (cancelled) return
        setRows(data.recipients ?? [])
        setTotal(data.total)
      } catch (err) {
        if (cancelled) return
        setError(err instanceof Error ? err.message : "Failed to load recipients")
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [campaignId, status, offset])

  const changeStatus = (next: RecipientStatus | "") => {
    setStatus(next)
    setOffset(0)
  }

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <select
          value={status}
          onChange={(e) => changeStatus(e.target.value as RecipientStatus | "")}
          className="rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
        >
          <option value="">All statuses</option>
          <option value="sent">Delivered</option>
          <option value="failed">Failed</option>
          <option value="queued">Queued</option>
        </select>
        <p className="text-xs text-vn-text-3 tabular-nums">
          {total === 0
            ? "No recipients"
            : `${offset + 1}–${Math.min(offset + PAGE_SIZE, total)} of ${total}`}
        </p>
      </div>

      {error && <p className="text-sm text-vn-danger">{error}</p>}

      {isLoading && rows.length === 0 ? (
        <div className="flex items-center gap-2 rounded-xl border border-vn-hairline bg-vn-surface p-6 text-sm text-vn-text-3">
          <Loader2 className="size-4 animate-spin" />
          Loading recipients…
        </div>
      ) : rows.length === 0 ? (
        <EmptyPanel message="No recipients match this filter." />
      ) : (
        <div className="overflow-x-auto rounded-xl border border-vn-hairline bg-vn-surface">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Recipient</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Sent</TableHead>
                <TableHead className="text-right">Opens</TableHead>
                <TableHead>First open</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={r.email}>
                  <TableCell className="text-vn-text">{r.email}</TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <RecipientStatusBadge status={r.status} />
                      {r.error_message && (
                        <span className="max-w-xs break-words text-xs text-vn-danger">
                          {r.error_message}
                        </span>
                      )}
                    </div>
                  </TableCell>
                  <TableCell className="text-vn-text-3">
                    {formatDateTime(r.sent_at)}
                  </TableCell>
                  <TableCell className="text-right tabular-nums text-vn-text-2">
                    {r.open_count}
                  </TableCell>
                  <TableCell className="text-vn-text-3">
                    {formatDateTime(r.first_opened_at)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {total > PAGE_SIZE && (
        <div className="flex items-center justify-end gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={offset === 0 || isLoading}
            onClick={() => setOffset(Math.max(0, offset - PAGE_SIZE))}
          >
            Previous
          </Button>
          <Button
            variant="outline"
            size="sm"
            disabled={offset + PAGE_SIZE >= total || isLoading}
            onClick={() => setOffset(offset + PAGE_SIZE)}
          >
            Next
          </Button>
        </div>
      )}
    </div>
  )
}
