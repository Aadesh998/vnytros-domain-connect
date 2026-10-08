import { useMemo, useState } from "react"
import { Link } from "react-router-dom"
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
import { CheckCircle2, Eye, Loader2, Mail, MailX, Send } from "lucide-react"
import { useMailAnalytics } from "@/hooks/useMailAnalytics"
import { StatTile } from "./StatTile"
import { formatPercent } from "./format"

export function MailOverviewFeature() {
  const [days, setDays] = useState(30)
  const { analytics, isLoading, error } = useMailAnalytics(days)

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

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold text-vn-text">Mail</h1>
          <p className="mt-1 text-sm text-vn-text-3">
            How your campaigns are performing across the account.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            className="rounded-lg border border-vn-hairline-2 bg-vn-surface-2 px-3 py-2 text-sm text-vn-text focus:outline-none focus:ring-2 focus:ring-vn-accent/35"
          >
            <option value={7}>Last 7 days</option>
            <option value={30}>Last 30 days</option>
            <option value={90}>Last 90 days</option>
          </select>
          <Button asChild>
            <Link to="/mail/campaigns">
              <Send className="size-4" />
              Campaigns
            </Link>
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
          Loading analytics…
        </div>
      ) : analytics && analytics.total_campaigns === 0 ? (
        <div className="rounded-xl border border-vn-hairline bg-vn-surface p-8 text-center">
          <Mail className="mx-auto size-6 text-vn-text-4" />
          <p className="mt-3 text-sm font-medium text-vn-text">
            No campaigns yet
          </p>
          <p className="mt-1 text-sm text-vn-text-3">
            Create a template, then a campaign, and your figures appear here.
          </p>
          <div className="mt-4 flex justify-center gap-2">
            <Button variant="outline" asChild>
              <Link to="/mail/templates">New template</Link>
            </Button>
            <Button asChild>
              <Link to="/mail/campaigns">New campaign</Link>
            </Button>
          </div>
        </div>
      ) : analytics ? (
        <>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <StatTile
              label="Campaigns"
              value={analytics.total_campaigns}
              hint={
                analytics.active_campaigns > 0
                  ? `${analytics.active_campaigns} sending now`
                  : "None in flight"
              }
              icon={Mail}
            />
            <StatTile
              label="Delivered"
              value={analytics.sent_emails}
              hint={`${formatPercent(analytics.delivery_rate)} of ${analytics.total_emails} attempted`}
              icon={CheckCircle2}
              tone="positive"
            />
            <StatTile
              label="Failed"
              value={analytics.failed_emails}
              hint={formatPercent(analytics.failure_rate)}
              icon={MailX}
              tone={analytics.failed_emails > 0 ? "negative" : "neutral"}
            />
            <StatTile
              label="Unique opens"
              value={analytics.unique_opens}
              hint={`${formatPercent(analytics.open_rate)} open rate`}
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
                      <linearGradient
                        id="accountOpenFill"
                        x1="0"
                        y1="0"
                        x2="0"
                        y2="1"
                      >
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
                      fill="url(#accountOpenFill)"
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
        </>
      ) : null}
    </div>
  )
}
